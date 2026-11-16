import { getQueryClient } from "@/shared/lib/queryClient";
import { confirmAIAction } from "@/shared/lib/aiCredits";
import i18n from "i18next";
import ky, {
  type KyInstance,
  type NormalizedOptions,
  HTTPError,
  TimeoutError,
} from "ky";
import type { ErrorResponse } from "@/shared/types/api";
import { sessionSignal } from "@/shared/lib/sessionLifetime";
import { endSession } from "@/shared/lib/session";
import {
  type SessionGeneration,
  isSameAccountSession,
  isSessionGenerationCurrent,
  isTabSessionCurrent,
  sharedSessionGeneration,
  startSessionGeneration,
  syncTabSession,
  tabAccount,
  tabSessionGeneration,
} from "@/shared/lib/sessionGeneration";
import { loginPathFor } from "@/shared/lib/loginReturn";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

const REQUEST_TIMEOUT_MS = 30_000;
const AUTH_COOKIE_LOCK = "cursopus-auth-cookies";
const AUTH_LOCK_WAIT_TIMEOUT_MS = REQUEST_TIMEOUT_MS * 2;

async function withAuthCookieLock<T>(operation: () => Promise<T>, signal?: AbortSignal): Promise<T> {
  // The browser releases Web Locks when a tab closes or reloads. Browsers
  // without this API retain the per-client logout barrier below.
  if (typeof navigator === "undefined" || !navigator.locks?.request) return operation();
  const waiting = new AbortController();
  const timer = setTimeout(() => waiting.abort(new DOMException("Auth lock timed out", "TimeoutError")), AUTH_LOCK_WAIT_TIMEOUT_MS);
  try {
    return await navigator.locks.request(AUTH_COOKIE_LOCK, {
      signal: signal ? AbortSignal.any([signal, waiting.signal]) : waiting.signal,
    }, async () => {
      clearTimeout(timer);
      signal?.throwIfAborted();
      // Keep the lock until the transport settles or ky aborts it on timeout.
      // Releasing it by racing a timer would leave a late Set-Cookie alive.
      return operation();
    });
  } finally {
    clearTimeout(timer);
  }
}

function sessionChanged(): DOMException {
  return new DOMException("Session changed", "AbortError");
}

/**
 * Stops work that belongs to a session another tab has since ended or
 * replaced. This tab is signed out too: its state belongs to that session.
 * `undefined` is a public request, which belongs to no session.
 */
function assertSessionCurrent(generation: SessionGeneration | undefined): void {
  if (generation === undefined || isSessionGenerationCurrent(generation)) return;
  syncTabSession();
  throw sessionChanged();
}

/**
 * For a refresh and its replay: the session may also have been renewed by the
 * same account signing in again in another tab. The cookies are still that
 * account's, so its own request may go ahead.
 */
function assertSameAccount(
  generation: SessionGeneration | undefined,
  account: string | null | undefined,
): void {
  if (generation === undefined || isSameAccountSession(generation, account)) return;
  syncTabSession();
  throw sessionChanged();
}

/** The account a login response signed in, if it says. */
function signedInAccount(result: unknown): string | null {
  const id = (result as { user?: { id?: unknown } } | null)?.user?.id;
  return typeof id === "string" && id ? id : null;
}

/** File uploads and PDF renders need longer than an ordinary JSON call. */
const LONG_REQUEST_TIMEOUT_MS = 60_000;

/**
 * A credit-charging AI call may run for up to 90s on the backend
 * (AIRequestScope); nginx waits 95s and Cloudflare 100s for it. The client
 * waits longest — its clock also covers the upload, which the others don't
 * count — so a slow call ends with a server answer, not with an abort that
 * leaves a charged provider call running and invites a second, separately
 * charged try. Raising the server-side limits past Cloudflare's 100s is moot.
 */
const AI_REQUEST_TIMEOUT_MS = 120_000;

/** A caller's own budget may lengthen a charged AI call, never cut it short. */
function aiAwareTimeout(operationKey: string | undefined, fallbackMs: number, requestedMs?: number): number {
  if (!operationKey) return requestedMs ?? fallbackMs;
  return Math.max(requestedMs ?? 0, AI_REQUEST_TIMEOUT_MS);
}

/**
 * Failure codes this client produces itself, for everything that never reached
 * the API. Anything the server answered carries the server's own `error_code`.
 */
export const CLIENT_ERROR_CODES = {
  /** The request never completed — offline, DNS, CORS, a killed connection. */
  network: "NETWORK_ERROR",
  /** The request ran out of its own time budget. */
  timeout: "TIMEOUT_ERROR",
  /** A 2xx whose body was not the JSON the caller asked for. */
  invalidResponse: "INVALID_RESPONSE",
  /** Anything else — a bug on this side of the wire. */
  unexpected: "UNEXPECTED_ERROR",
} as const;

/** Everything a hook needs to replay a request after a token refresh. */
interface RequestContext {
  readonly timeoutMs: number;
  readonly session?: AbortSignal;
  /** The cross-tab session the request was sent under; absent on public paths. */
  readonly generation?: SessionGeneration;
  /** The account the tab showed when the request was sent. */
  readonly account?: string | null;
}

/**
 * Per-call options every public method accepts.
 *
 * `signal` is the caller's own cancellation — React Query hands one to every
 * `queryFn` and aborts it when the component unmounts or the key changes, and
 * a hand-rolled `AbortController` works the same way. It is carried onto the
 * 401 replay as well, so a request that survives a token refresh is still
 * cancellable; before there was no way to pass one in at all, and the replay's
 * careful signal combination had nothing but its own timeout to combine.
 */
export interface RequestOptions {
  readonly signal?: AbortSignal;
  /** Origin lifetime for consent only; it never aborts an already dispatched request. */
  readonly consentSignal?: AbortSignal;
}

/**
 * Sends a user whose session just ended to a login screen that brings them
 * back to the page they were on — but only when
 * they are inside the authenticated app. A public page (landing, blog, the
 * login route itself) needs no redirect, which also rules out a loop: `/login`
 * is public, so it never redirects to itself.
 */
function redirectToLogin(): void {
  const path = window.location.pathname ?? "";
  if (path === "/app" || path.startsWith("/app/")) {
    window.location.href = loginPathFor(path + (window.location.search ?? ""));
  }
}

function contextTimeout(options: NormalizedOptions): number {
  const context = options.context as Partial<RequestContext> | undefined;
  return typeof context?.timeoutMs === "number"
    ? context.timeoutMs
    : REQUEST_TIMEOUT_MS;
}

class ApiClient {
  private client: KyInstance;
  private refreshPromise: {
    signal: AbortSignal;
    generation: SessionGeneration | undefined;
    promise: Promise<boolean>;
  } | null = null;
  private logoutRequest: Promise<unknown> | null = null;

  constructor() {
    this.client = ky.create({
      prefixUrl: API_BASE_URL,
      timeout: REQUEST_TIMEOUT_MS,
      credentials: "include",
      hooks: {
        beforeRequest: [
          (request) => {
            // Add request_id for tracing
            const requestId = this.generateRequestId();
            request.headers.set("X-Request-ID", requestId);

            return request;
          },
        ],
        afterResponse: [
          async (request, options, response) => {
            // Handle 401 Unauthorized - try to refresh token via cookie
            // Skip for auth endpoints (login/register return 401 on bad credentials)
            const url = new URL(request.url);
            const isAuthEndpoint = url.pathname.includes("/auth/");

            const context = options.context as Partial<RequestContext>;
            context.session?.throwIfAborted();
            if (response.status !== 401 || isAuthEndpoint) {
              return response;
            }

            const lifetime = context.session;
            lifetime?.throwIfAborted();
            // A 401 for a session another tab has ended is no reason to
            // refresh: the cookies may belong to someone else by now.
            assertSameAccount(context.generation, context.account);
            const refreshed = await this.tryRefreshToken(lifetime ?? sessionSignal(), context.generation, context.account);
            lifetime?.throwIfAborted();
            if (!refreshed) {
              // The session is over. Everything cached under it goes with it —
              // a reload would rebuild the cache anyway, but the seconds before
              // it were long enough to paint the previous account's data.
              endSession();
              redirectToLogin();
              return response;
            }

            // Another tab may have logged out or signed in while the refresh
            // waited: the replay would carry this request's body into it.
            assertSameAccount(context.generation, context.account);
            return this.retryOnce(request, contextTimeout(options));
          },
        ],
      },
    });
  }

  /**
   * Replays a request exactly once after a successful token refresh.
   *
   * Four things this must not do, each of which broke the first action after
   * a token expiry before:
   *
   *  - add a header. The retry used to carry an `X-Retry: 1` marker, which is
   *    not in the API's `Access-Control-Allow-Headers`, so the browser failed
   *    the CORS preflight and the replayed request never reached the server —
   *    surfacing as a bare "Failed to fetch".
   *  - inherit the first attempt's timeout. `new Request(original)` carries the
   *    original signal over, and part of what that signal represents is a
   *    budget the first attempt has already spent; a late abort would kill the
   *    replay mid-flight. The retry gets a fresh timeout of its own.
   *
   *    What it must *not* drop along with that budget is the caller's own
   *    cancellation. The signal ky puts on the request is a combination —
   *    ky's timeout and whatever `signal` the caller passed — so replacing it
   *    outright left a replayed request that no unmount, navigation or
   *    `AbortController` could stop. The two are combined instead.
   *  - shorten that timeout. It used to be hard-coded at the 30s default, so a
   *    60s upload or PDF render came back from a token refresh with half its
   *    budget — a large file that had just been given a minute was cut off at
   *    thirty seconds, and the customer saw a bare failure. The original
   *    request's own budget is carried through `options.context` and used here.
   *  - loop. The replay goes through bare `ky` — no `afterResponse` hook, no
   *    internal retries — so a second 401 is returned as-is rather than
   *    triggering another refresh.
   *
   * `request` is ky's untouched clone of the original, so its body is still
   * readable and the replay is byte-identical to what the server rejected.
   * The server already refused the first attempt, so nothing is submitted
   * twice.
   */
  private async retryOnce(
    request: Request,
    timeoutMs: number,
  ): Promise<Response> {
    const retryRequest = new Request(request, {
      credentials: "include",
      signal: AbortSignal.any([request.signal, AbortSignal.timeout(timeoutMs)]),
    });

    // throwHttpErrors: false — a non-2xx replay is handed back to ky, which
    // raises the HTTPError with the response body intact so handleError can
    // read the API's own error code and message out of it.
    return ky(retryRequest, {
      retry: 0,
      timeout: false,
      throwHttpErrors: false,
    });
  }

  private generateRequestId(): string {
    return crypto.randomUUID();
  }

  private async tryRefreshToken(
    signal: AbortSignal,
    generation: SessionGeneration | undefined,
    account: string | null | undefined,
  ): Promise<boolean> {
    signal.throwIfAborted();
    const pending = this.refreshPromise;
    if (pending?.signal === signal && pending.generation === generation) return pending.promise;
    const refresh = { signal, generation, promise: this.doRefreshToken(signal, generation, account) };
    this.refreshPromise = refresh;
    try {
      return await refresh.promise;
    } finally {
      if (this.refreshPromise === refresh) this.refreshPromise = null;
    }
  }

  private async doRefreshToken(
    signal: AbortSignal,
    generation: SessionGeneration | undefined,
    account: string | null | undefined,
  ): Promise<boolean> {
    try {
      await this.waitForLogout();
      await withAuthCookieLock(async () => {
        signal.throwIfAborted();
        assertSameAccount(generation, account);
        await ky.post(`${API_BASE_URL}/auth/refresh`, {
          credentials: "include", signal, retry: 0, timeout: REQUEST_TIMEOUT_MS,
        });
      }, signal);
      signal.throwIfAborted();
      return true;
    } catch (err) {
      signal.throwIfAborted();
      // Not a refused refresh: the session moved on, and ending "it" here
      // would sign out whichever session replaced it.
      assertSameAccount(generation, account);
      console.error("[ApiClient] token refresh failed:", err instanceof Error ? err.message : "unknown error");
      return false;
    }
  }

  /**
   * Normalises every transport failure into an ApiError.
   *
   * The only text allowed through to `message` is the API's own
   * `error_message`, which is written to be read by a customer. Everything the
   * HTTP client produces — "Request failed with status code 404 Not Found:
   * POST http://…", "Failed to fetch" — is dropped: it used to reach toasts
   * verbatim, leaking the status line and the raw backend URL. An empty message
   * is deliberate; call sites read `error.message || t("…")` and fall back to
   * their own localized copy, and `showErrorNotification` has a generic
   * fallback for anything that slips through.
   *
   * What the empty message must not cost is the *diagnosis*. Every failure now
   * carries a code that says which kind it was and the original error as
   * `cause`, so a timeout is distinguishable from an unreachable server and a
   * bug on this side is distinguishable from either. The two that mean "this
   * code is wrong" — a 2xx that is not the JSON the caller asked for, and
   * anything unrecognised — are logged, with only the operation and the error's
   * own name and message: never the request body, headers or cookies.
   */
  private async handleError(error: unknown, operation: string): Promise<never> {
    if (error instanceof HTTPError) {
      const status = error.response.status;
      let message = "";
      let code = "UNKNOWN_ERROR";
      let unstructuredResponse = false;
      try {
        const errorResponse = await error.response.json<ErrorResponse>();
        message = errorResponse?.error_message ?? "";
        code = errorResponse?.error_code || code;
      } catch {
        // Non-JSON body (gateway/HTML error page): nothing quotable.
        unstructuredResponse = true;
      }
      const creditErrors: Record<string, string> = {
        AI_JOB_DESCRIPTION_REQUIRED: "coverLetter.ai.jobDescriptionRequired",
        AI_CANDIDATE_SOURCE_REQUIRED: "coverLetter.ai.resumeRequired",
        AI_NO_CHANGES: "common.aiErrors.noChanges",
        AI_RESULT_SAVE_FAILED: "common.aiErrors.resultSaveFailed",
        INVALID_DATE_RANGE: "resumeBuilder.errors.invalidDateRange",
        AI_INVALID_RESPONSE: "ai.invalidResponse",
        CREDIT_UNAVAILABLE: "credits.exhausted",
        AI_BUDGET_REACHED: "credits.budgetReached",
        AI_OPERATION_CONFLICT: "credits.operationConflict",
      };
      if (creditErrors[code]) message = i18n.t(creditErrors[code]);
      throw new ApiError(message, code, status, { cause: error, unstructuredResponse });
    }

    // A caller-cancelled request (a React Query unmount, a navigation, an
    // `AbortController` a caller passed in) is not a failure to report — hand
    // it back untouched so it stays a cancellation.
    //
    // Matched on the name rather than on `instanceof DOMException`. The class
    // is realm-bound: an abort raised inside `fetch` comes from the platform's
    // own realm, which is not the one this module's `DOMException` binding
    // points at under a jsdom test runner, and the check quietly fell through
    // to "a bug on this side" — logging a cancellation and reporting it as
    // UNEXPECTED_ERROR. The name is the part of the contract that is stable.
    if (errorName(error) === "AbortError") {
      throw error;
    }

    // ky raises its own TimeoutError; the refresh replay's AbortSignal.timeout
    // raises a DOMException of the same name. Both mean the budget ran out,
    // which is a different thing from an unreachable server: the request may
    // well have been received and acted on.
    if (error instanceof TimeoutError || errorName(error) === "TimeoutError") {
      throw new ApiError("", CLIENT_ERROR_CODES.timeout, 0, { cause: error });
    }

    // fetch reports every transport failure — offline, DNS, CORS, a dropped
    // connection — as a TypeError and nothing else.
    if (error instanceof TypeError) {
      throw new ApiError("", CLIENT_ERROR_CODES.network, 0, { cause: error });
    }

    // The response arrived and was a success, but its body was not the JSON
    // the caller asked for. That is the API or a proxy misbehaving, not the
    // network.
    if (error instanceof SyntaxError) {
      logClientFailure(operation, error);
      throw new ApiError("", CLIENT_ERROR_CODES.invalidResponse, 0, {
        cause: error,
      });
    }

    // Anything left is a bug on this side — a throwing hook, a bad option.
    // Swallowing it as NETWORK_ERROR is what made those invisible.
    logClientFailure(operation, error);
    throw new ApiError("", CLIENT_ERROR_CODES.unexpected, 0, { cause: error });
  }

  async get<T>(url: string, options?: RequestOptions): Promise<T> {
    try {
      return await this.client
        .get(url, requestOptions(REQUEST_TIMEOUT_MS, options, url))
        .json<T>();
    } catch (error) {
      return this.handleError(error, `GET ${operationPath(url)}`);
    }
  }

  async post<T>(
    url: string,
    data?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    const path = operationPath(url);
    if (path === "auth/logout") {
      // Share one logout response: a second late Set-Cookie could otherwise
      // erase the next session too. Register it before any asynchronous work.
      if (this.logoutRequest) return this.logoutRequest as Promise<T>;
      // Another tab already ended the session this tab is logging out of. The
      // cookies may belong to a newer sign-in now; leave them alone. (A tab
      // whose account signed in again elsewhere has adopted it by now: on the
      // broadcast, when shown, or in authService.logout before its state goes.)
      if (!isTabSessionCurrent()) {
        syncTabSession();
        throw sessionChanged();
      }
      // A new generation at once: sign-ins and refreshes started before this
      // logout, in any tab, must not land after it.
      const logout = this.postAuthRequest<T>(url, data, options, startSessionGeneration());
      this.logoutRequest = logout;
      try {
        return await logout;
      } finally {
        this.logoutRequest = null;
      }
    }
    if (/^auth\/(login|register|refresh)$/.test(path)) {
      // These start from whatever session is current, even if this tab has
      // not caught up with it yet; a logout or sign-in after this point wins.
      const generation = sharedSessionGeneration();
      await this.waitForLogout();
      return this.postAuthRequest<T>(url, data, options, generation);
    }
    return this.postRequest<T>(url, data, options);
  }

  private async postAuthRequest<T>(
    url: string,
    data: unknown,
    options: RequestOptions | undefined,
    generation: SessionGeneration,
  ): Promise<T> {
    const path = operationPath(url);
    try {
      return await withAuthCookieLock(async () => {
        // Checked once the lock is held, right before sending: whoever held it
        // last may have logged out or signed in another account.
        assertSessionCurrent(generation);
        if (path !== "auth/logout") {
          const result = await this.postRequest<T>(url, data, options);
          // Still inside the lock, so the next queued operation in any tab
          // already sees the new session. Only login sets session cookies;
          // register and refresh leave the session as it was.
          if (path === "auth/login") {
            assertSessionCurrent(generation);
            startSessionGeneration(signedInAccount(result));
          }
          return result;
        }
        // ky's timeout ends at headers. Keep an aborting deadline through body
        // parsing (including error bodies), so a stalled stream releases the lock.
        const deadline = new AbortController();
        const timer = setTimeout(() => deadline.abort(new DOMException("Logout timed out", "TimeoutError")), REQUEST_TIMEOUT_MS);
        try {
          return await this.postRequest<T>(url, data, {
            ...options,
            signal: options?.signal ? AbortSignal.any([options.signal, deadline.signal]) : deadline.signal,
          });
        } catch (error) {
          deadline.signal.throwIfAborted();
          throw error;
        } finally {
          clearTimeout(timer);
        }
      }, options?.signal);
    } catch (error) {
      if (error instanceof ApiError) throw error;
      return this.handleError(error, `POST ${path}`);
    }
  }

  private async waitForLogout(): Promise<void> {
    // Browsers apply Set-Cookie before JS sees the response. Session generation
    // checks cannot protect a new login from an older logout's cookie deletion.
    // Wait for the actual request to settle (including ky's abort on timeout),
    // rather than racing a timer that leaves the cookie-clearing fetch alive.
    while (this.logoutRequest) {
      // The logout caller reports failures; a settled failure must allow login.
      await this.logoutRequest.then(() => undefined, () => undefined);
    }
  }

  private async postRequest<T>(
    url: string,
    data?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    const scopedOptions = requestOptions(REQUEST_TIMEOUT_MS, options, url);
    const operationKey = await confirmAIAction(url, consentSignal({ ...options, signal: scopedOptions.signal }));
    const timeoutMs = aiAwareTimeout(operationKey, REQUEST_TIMEOUT_MS);
    try {
      return await this.client
        .post(url, { json: data, ...scopedOptions, timeout: timeoutMs,
          context: { ...scopedOptions.context, timeoutMs },
          ...(operationKey ? { headers: { "Idempotency-Key": operationKey }, retry: 0 } : {}),
        })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `POST ${operationPath(url)}`);
    } finally {
      if (operationKey) void getQueryClient().invalidateQueries({ queryKey: ["subscription"] });
    }
  }

  async patch<T>(
    url: string,
    data?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    try {
      return await this.client
        .patch(url, {
          json: data,
          ...requestOptions(REQUEST_TIMEOUT_MS, options, url),
        })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `PATCH ${operationPath(url)}`);
    }
  }

  async put<T>(
    url: string,
    data?: unknown,
    options?: RequestOptions,
  ): Promise<T> {
    try {
      return await this.client
        .put(url, { json: data, ...requestOptions(REQUEST_TIMEOUT_MS, options, url) })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `PUT ${operationPath(url)}`);
    }
  }

  async delete<T>(url: string, options?: RequestOptions): Promise<T> {
    try {
      return await this.client
        .delete(url, requestOptions(REQUEST_TIMEOUT_MS, options, url))
        .json<T>();
    } catch (error) {
      return this.handleError(error, `DELETE ${operationPath(url)}`);
    }
  }

  async postBlob(
    url: string,
    data?: unknown,
    timeout?: number,
    options?: RequestOptions,
  ): Promise<Blob> {
    const timeoutMs = timeout ?? LONG_REQUEST_TIMEOUT_MS;
    try {
      const response = await this.client.post(url, {
        json: data,
        ...requestOptions(timeoutMs, options, url),
      });
      return await response.blob();
    } catch (error) {
      return this.handleError(error, `POST ${operationPath(url)}`);
    }
  }

  async postFormData<T>(
    url: string,
    formData: FormData,
    timeout?: number,
    options?: RequestOptions,
  ): Promise<T> {
    const scopedOptions = requestOptions(REQUEST_TIMEOUT_MS, options, url);
    const operationKey = await confirmAIAction(url, consentSignal({ ...options, signal: scopedOptions.signal }));
    const timeoutMs = aiAwareTimeout(operationKey, LONG_REQUEST_TIMEOUT_MS, timeout);
    try {
      return await this.client
        .post(url, { body: formData, ...scopedOptions, timeout: timeoutMs, context: { ...scopedOptions.context, timeoutMs },
          ...(operationKey ? { headers: { "Idempotency-Key": operationKey }, retry: 0 } : {}),
        })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `POST ${operationPath(url)}`);
    } finally {
      if (operationKey) void getQueryClient().invalidateQueries({ queryKey: ["subscription"] });
    }
  }
}

function consentSignal(options?: RequestOptions): AbortSignal | undefined {
  if (options?.signal && options.consentSignal) return AbortSignal.any([options.signal, options.consentSignal]);
  return options?.consentSignal ?? options?.signal;
}

/**
 * ky options for one request: the timeout, and the same number again in
 * `context` so the 401 replay can find it. `context` is the only channel ky
 * gives a hook, and without it the replay fell back to the 30s default and cut
 * a 60s upload in half.
 */
function requestOptions(timeoutMs: number, options: RequestOptions | undefined, url: string) {
  const publicPath = /^(auth\/|public\/|blog(?:\/|$)|features\/|subscription\/checkout-config(?:[?]|$))/;
  const isPublic = publicPath.test(url);
  const lifetime = isPublic ? undefined : sessionSignal();
  const signal = lifetime && options?.signal
    ? AbortSignal.any([lifetime, options.signal])
    : lifetime ?? options?.signal;
  return {
    timeout: timeoutMs, signal,
    context: {
      timeoutMs, session: lifetime,
      generation: isPublic ? undefined : tabSessionGeneration(),
      account: isPublic ? undefined : tabAccount(),
    } satisfies RequestContext,
  };
}

/**
 * The part of a request path that is safe to log: no query string, which is
 * where search terms and other customer text live.
 */
function operationPath(url: string): string {
  return url.split("?")[0];
}

/**
 * The `name` a thrown value carries, or "" when it carries none.
 *
 * Used instead of `instanceof` for the platform's own error classes, which are
 * per-realm: the same abort is a `DOMException` to `fetch` and not one to this
 * module when the two are bound to different globals.
 */
function errorName(error: unknown): string {
  if (typeof error !== "object" || error === null) return "";
  const { name } = error as { name?: unknown };
  return typeof name === "string" ? name : "";
}

/** Logs a failure that means this code is wrong, without quoting any payload. */
function logClientFailure(operation: string, error: unknown): void {
  const name = error instanceof Error ? error.name : typeof error;
  const message = error instanceof Error ? error.message : "";
  console.error(`[ApiClient] ${operation} failed: ${name}: ${message}`);
}

export class ApiError extends Error {
  code: string;
  status: number;
  readonly unstructuredResponse: boolean;

  constructor(
    message: string,
    code: string,
    status: number,
    options?: ErrorOptions & { unstructuredResponse?: boolean },
  ) {
    super(message, options);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.unstructuredResponse = options?.unstructuredResponse ?? false;
  }
}

export const apiClient = new ApiClient();
