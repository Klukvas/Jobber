import ky, {
  type KyInstance,
  type NormalizedOptions,
  HTTPError,
  TimeoutError,
} from "ky";
import type { ErrorResponse } from "@/shared/types/api";
import { endSession } from "@/shared/lib/session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

const REQUEST_TIMEOUT_MS = 30_000;

/** File uploads and PDF renders need longer than an ordinary JSON call. */
const LONG_REQUEST_TIMEOUT_MS = 60_000;

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
}

function contextTimeout(options: NormalizedOptions): number {
  const context = options.context as Partial<RequestContext> | undefined;
  return typeof context?.timeoutMs === "number"
    ? context.timeoutMs
    : REQUEST_TIMEOUT_MS;
}

class ApiClient {
  private client: KyInstance;
  private refreshPromise: Promise<boolean> | null = null;

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

            if (response.status !== 401 || isAuthEndpoint) {
              return response;
            }

            const refreshed = await this.tryRefreshToken();
            if (!refreshed) {
              // The session is over. Everything cached under it goes with it —
              // a reload would rebuild the cache anyway, but the seconds before
              // it were long enough to paint the previous account's data.
              endSession();
              window.location.href = "/";
              return response;
            }

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

  private async tryRefreshToken(): Promise<boolean> {
    // Deduplicate concurrent refresh attempts
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefreshToken();
    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefreshToken(): Promise<boolean> {
    try {
      // Refresh token is sent automatically via httpOnly cookie
      await ky.post(`${API_BASE_URL}/auth/refresh`, {
        credentials: "include",
      });
      return true;
    } catch (err) {
      console.error(
        "[ApiClient] token refresh failed:",
        err instanceof Error ? err.message : "unknown error",
      );
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
      try {
        const errorResponse = await error.response.json<ErrorResponse>();
        message = errorResponse?.error_message ?? "";
        code = errorResponse?.error_code || code;
      } catch {
        // Non-JSON body (gateway/HTML error page): nothing quotable.
      }
      throw new ApiError(message, code, status, { cause: error });
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
        .get(url, requestOptions(REQUEST_TIMEOUT_MS, options))
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
    try {
      return await this.client
        .post(url, { json: data, ...requestOptions(REQUEST_TIMEOUT_MS, options) })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `POST ${operationPath(url)}`);
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
          ...requestOptions(REQUEST_TIMEOUT_MS, options),
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
        .put(url, { json: data, ...requestOptions(REQUEST_TIMEOUT_MS, options) })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `PUT ${operationPath(url)}`);
    }
  }

  async delete<T>(url: string, options?: RequestOptions): Promise<T> {
    try {
      return await this.client
        .delete(url, requestOptions(REQUEST_TIMEOUT_MS, options))
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
        ...requestOptions(timeoutMs, options),
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
    const timeoutMs = timeout ?? LONG_REQUEST_TIMEOUT_MS;
    try {
      return await this.client
        .post(url, { body: formData, ...requestOptions(timeoutMs, options) })
        .json<T>();
    } catch (error) {
      return this.handleError(error, `POST ${operationPath(url)}`);
    }
  }
}

/**
 * ky options for one request: the timeout, and the same number again in
 * `context` so the 401 replay can find it. `context` is the only channel ky
 * gives a hook, and without it the replay fell back to the 30s default and cut
 * a 60s upload in half.
 */
function requestOptions(timeoutMs: number, options?: RequestOptions) {
  return {
    timeout: timeoutMs,
    signal: options?.signal,
    context: { timeoutMs } satisfies RequestContext,
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

  constructor(
    message: string,
    code: string,
    status: number,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

export const apiClient = new ApiClient();
