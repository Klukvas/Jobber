import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const clearAuth = vi.fn();

vi.mock("@/stores/authStore", () => ({
  useAuthStore: { getState: () => ({ clearAuth }) },
}));

const BASE = "http://api.test/api/v1";

type ApiModule = typeof import("../api");

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

/**
 * Regression coverage for the first request after an access token expires.
 * The retry must reach the server (no unexpected headers), must run exactly
 * once, and must not be tied to the first attempt's AbortSignal.
 */
describe("apiClient 401 refresh-and-retry", () => {
  let api: ApiModule;
  let requests: Request[];
  let bodies: string[];
  let originalFetch: typeof globalThis.fetch;
  let originalLocation: Location;

  beforeEach(async () => {
    clearAuth.mockClear();
    requests = [];
    bodies = [];
    originalFetch = globalThis.fetch;
    originalLocation = window.location;
    Object.defineProperty(window, "location", {
      value: { href: "http://app.test/app/jobs" },
      writable: true,
      configurable: true,
    });

    vi.stubEnv("VITE_API_BASE_URL", BASE);
    vi.resetModules();
    api = await import("../api");
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    Object.defineProperty(window, "location", {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
    vi.unstubAllEnvs();
  });

  /**
   * A response that never arrives on its own and honours cancellation exactly
   * as `fetch` does — including a signal that was already aborted before the
   * request was dispatched.
   */
  function neverAnswers(request: Request): Promise<Response> {
    return new Promise<Response>((_resolve, reject) => {
      const cancel = () =>
        reject(
          request.signal.reason ?? new DOMException("aborted", "AbortError"),
        );
      if (request.signal.aborted) {
        cancel();
        return;
      }
      request.signal.addEventListener("abort", cancel);
    });
  }

  function installFetch(
    handler: (request: Request) => Response | Promise<Response>,
  ) {
    globalThis.fetch = vi.fn(async (input: RequestInfo | URL) => {
      const request = input as Request;
      requests.push(request);
      // Read from a clone: ky consumes (and later cancels) the real body, so
      // it is unusable by the time the assertions run.
      bodies.push(request.body ? await request.clone().text() : "");
      return handler(request);
    }) as unknown as typeof globalThis.fetch;
  }

  it("refreshes once and replays the original request exactly once", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1
        ? jsonResponse({ error_code: "UNAUTHORIZED" }, 401)
        : jsonResponse({ id: "job-1" });
    });

    const result = await api.apiClient.patch<{ id: string }>("jobs/job-1", {
      title: "renamed",
    });

    expect(result).toEqual({ id: "job-1" });
    expect(protectedCalls).toBe(2);

    const [first, refresh, retry] = requests;
    expect(first.method).toBe("PATCH");
    expect(refresh.url).toBe(`${BASE}/auth/refresh`);
    expect(retry.method).toBe("PATCH");
    expect(retry.url).toBe(first.url);
    expect(bodies[2]).toBe(JSON.stringify({ title: "renamed" }));
    expect(bodies[2]).toBe(bodies[0]);
  });

  it("sends no extra headers on the replay, so CORS preflight still passes", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1 ? jsonResponse({}, 401) : jsonResponse({});
    });

    await api.apiClient.post("jobs", { title: "x" });

    const [first, , retry] = requests;
    expect(retry.headers.get("X-Retry")).toBeNull();
    expect([...retry.headers.keys()].sort()).toEqual(
      [...first.headers.keys()].sort(),
    );
  });

  /**
   * The replay has to keep answering to the caller's own cancellation.
   *
   * ky hands the hook a signal that combines its timeout with whatever `signal`
   * the caller passed, so replacing that signal outright — which is what giving
   * the replay a fresh timeout used to do — left a request in flight that no
   * unmount, navigation or `AbortController` could stop. The replay's signal
   * has to be the two combined, not the budget alone.
   */
  it("keeps the replay answering to the original request's signal", async () => {
    // ky combines signals of its own, so the replay's combination is picked out
    // by the fresh timeout budget that only it creates.
    const budgets = new Set<AbortSignal>();
    const realTimeout = AbortSignal.timeout.bind(AbortSignal);
    const timeoutSpy = vi
      .spyOn(AbortSignal, "timeout")
      .mockImplementation((ms: number) => {
        const signal = realTimeout(ms);
        budgets.add(signal);
        return signal;
      });

    const combined: AbortSignal[][] = [];
    const realAny = AbortSignal.any.bind(AbortSignal);
    const anySpy = vi
      .spyOn(AbortSignal, "any")
      .mockImplementation((signals: Iterable<AbortSignal>) => {
        const list = [...signals];
        combined.push(list);
        return realAny(list);
      });

    try {
      let protectedCalls = 0;
      installFetch((request) => {
        if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
        protectedCalls += 1;
        return protectedCalls === 1 ? jsonResponse({}, 401) : jsonResponse({});
      });

      await api.apiClient.get("jobs");

      const [, , retry] = requests;
      const replaySignals = combined.filter((list) =>
        list.some((signal) => budgets.has(signal)),
      );

      expect(replaySignals).toHaveLength(1);
      // A fresh budget *and* the signal the original request was carrying —
      // not the budget on its own, which is what dropped the cancellation.
      const [replay] = replaySignals;
      expect(replay.filter((signal) => budgets.has(signal))).toHaveLength(1);
      expect(replay.filter((signal) => !budgets.has(signal))).toHaveLength(1);
      expect(retry.signal.aborted).toBe(false);
    } finally {
      anySpy.mockRestore();
      timeoutSpy.mockRestore();
    }
  });

  it("gives the replay its own abort signal", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1 ? jsonResponse({}, 401) : jsonResponse({});
    });

    await api.apiClient.get("jobs");

    const [first, , retry] = requests;
    expect(retry.signal).not.toBe(first.signal);
    expect(retry.signal.aborted).toBe(false);
  });

  it("does not retry when the refresh itself fails, and clears auth", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({}, 401);
      protectedCalls += 1;
      return jsonResponse({ error_code: "UNAUTHORIZED" }, 401);
    });

    await expect(api.apiClient.get("jobs")).rejects.toBeInstanceOf(
      api.ApiError,
    );

    expect(protectedCalls).toBe(1);
    expect(clearAuth).toHaveBeenCalledTimes(1);
    expect(window.location.href).toBe("/");
  });

  it("surfaces the API error code when the replay is rejected again", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1
        ? jsonResponse({}, 401)
        : jsonResponse(
            {
              error_code: "PLAN_LIMIT_REACHED",
              error_message: "Limit reached",
            },
            403,
          );
    });

    await expect(api.apiClient.get("jobs")).rejects.toMatchObject({
      code: "PLAN_LIMIT_REACHED",
      message: "Limit reached",
      status: 403,
    });
  });

  it("never surfaces the transport's own message to the caller", async () => {
    globalThis.fetch = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    }) as unknown as typeof globalThis.fetch;

    await expect(api.apiClient.get("jobs")).rejects.toMatchObject({
      code: "NETWORK_ERROR",
      message: "",
    });
  });

  /**
   * Every tab and widget on a page fires its own request, so an expired token
   * produces a burst of 401s at once. One refresh has to serve all of them —
   * a refresh per request would rotate the token underneath its own siblings
   * and sign the customer out — and each request must be replayed once, not
   * once per sibling.
   */
  it("refreshes once for N simultaneous 401s and replays each exactly once", async () => {
    const PARALLEL = 5;
    let refreshCalls = 0;
    const attemptsByPath = new Map<string, number>();

    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return jsonResponse({});
      }
      const path = new URL(request.url).pathname;
      const attempts = (attemptsByPath.get(path) ?? 0) + 1;
      attemptsByPath.set(path, attempts);
      return attempts === 1
        ? jsonResponse({ error_code: "UNAUTHORIZED" }, 401)
        : jsonResponse({ path });
    });

    const results = await Promise.all(
      Array.from({ length: PARALLEL }, (_, index) =>
        api.apiClient.get<{ path: string }>(`jobs/job-${index}`),
      ),
    );

    expect(refreshCalls).toBe(1);
    expect(results.map((r) => r.path)).toEqual(
      Array.from({ length: PARALLEL }, (_, i) => `/api/v1/jobs/job-${i}`),
    );
    // Two attempts each: the rejected original and its single replay.
    for (const [path, attempts] of attemptsByPath) {
      expect(attempts, path).toBe(2);
    }
  });

  it("does not refresh again for a 401 that arrives after the first refresh finished", async () => {
    let refreshCalls = 0;
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) {
        refreshCalls += 1;
        return jsonResponse({});
      }
      protectedCalls += 1;
      return protectedCalls === 1
        ? jsonResponse({ error_code: "UNAUTHORIZED" }, 401)
        : jsonResponse({});
    });

    await api.apiClient.get("jobs");
    await api.apiClient.get("jobs");

    expect(refreshCalls).toBe(1);
  });

  /**
   * An upload's body is a multipart stream, not a string. The replay builds a
   * new Request from ky's clone, and if that clone's body did not survive the
   * first attempt the server would receive an empty upload after a token
   * refresh — a silent data loss rather than a visible failure.
   */
  it("replays a form-data upload with its body intact", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1
        ? jsonResponse({ error_code: "UNAUTHORIZED" }, 401)
        : jsonResponse({ id: "resume-1" });
    });

    const form = new FormData();
    form.append("title", "Backend Engineer");
    form.append(
      "file",
      new File(["%PDF-1.7"], "cv.pdf", { type: "application/pdf" }),
    );

    const result = await api.apiClient.postFormData<{ id: string }>(
      "resumes/upload",
      form,
    );

    expect(result).toEqual({ id: "resume-1" });
    expect(protectedCalls).toBe(2);
    // Byte-identical to the attempt the server rejected: the multipart
    // boundary, the text field and the file part all survived the replay.
    // (The file's *bytes* are not asserted — this environment's FormData
    // serializer does not inline a File's content — but an empty replay body
    // would still fail the equality and the two part assertions below.)
    expect(bodies[2]).toBe(bodies[0]);
    expect(bodies[2]).toContain("Backend Engineer");
    expect(bodies[2]).toContain('name="file"');
    expect(bodies[2].length).toBeGreaterThan(0);
  });

  it("replays a blob download and returns the replayed body", async () => {
    let protectedCalls = 0;
    installFetch((request) => {
      if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
      protectedCalls += 1;
      return protectedCalls === 1
        ? jsonResponse({ error_code: "UNAUTHORIZED" }, 401)
        : new Response("%PDF-1.7 rendered", {
            status: 200,
            headers: { "content-type": "application/pdf" },
          });
    });

    const blob = await api.apiClient.postBlob("resumes/r1/export", {
      format: "pdf",
    });

    expect(protectedCalls).toBe(2);
    expect(await blob.text()).toBe("%PDF-1.7 rendered");
    expect(bodies[2]).toBe(bodies[0]);
  });

  /**
   * The replay used to hard-code the 30s default. An upload or a PDF render is
   * given 60s, so a token refresh silently halved the budget of exactly the
   * requests that need it most.
   */
  describe("the replay keeps the original request's time budget", () => {
    function captureSignalTimeouts() {
      const timeouts: number[] = [];
      const original = AbortSignal.timeout.bind(AbortSignal);
      const spy = vi
        .spyOn(AbortSignal, "timeout")
        .mockImplementation((ms: number) => {
          timeouts.push(ms);
          return original(ms);
        });
      return { timeouts, restore: () => spy.mockRestore() };
    }

    function installReplayFetch() {
      let protectedCalls = 0;
      installFetch((request) => {
        if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
        protectedCalls += 1;
        return protectedCalls === 1 ? jsonResponse({}, 401) : jsonResponse({});
      });
    }

    it("gives a form-data upload its full 60s on the replay", async () => {
      const { timeouts, restore } = captureSignalTimeouts();
      try {
        installReplayFetch();
        const form = new FormData();
        form.append("file", new Blob(["x"]), "cv.pdf");

        await api.apiClient.postFormData("resumes/upload", form);

        expect(timeouts.at(-1)).toBe(60_000);
      } finally {
        restore();
      }
    });

    it("gives a blob render its full 60s on the replay", async () => {
      const { timeouts, restore } = captureSignalTimeouts();
      try {
        installReplayFetch();

        await api.apiClient.postBlob("resumes/r1/export", { format: "pdf" });

        expect(timeouts.at(-1)).toBe(60_000);
      } finally {
        restore();
      }
    });

    it("honours an explicitly requested budget", async () => {
      const { timeouts, restore } = captureSignalTimeouts();
      try {
        installReplayFetch();
        const form = new FormData();
        form.append("file", new Blob(["x"]), "cv.pdf");

        await api.apiClient.postFormData("resumes/upload", form, 90_000);

        expect(timeouts.at(-1)).toBe(90_000);
      } finally {
        restore();
      }
    });

    it("leaves an ordinary JSON call on the 30s default", async () => {
      const { timeouts, restore } = captureSignalTimeouts();
      try {
        installReplayFetch();

        await api.apiClient.get("jobs");

        expect(timeouts.at(-1)).toBe(30_000);
      } finally {
        restore();
      }
    });
  });

  /**
   * The replay combines its fresh budget with the caller's own signal so a
   * request that survives a token refresh is still cancellable. Until the
   * public methods took a signal there was no way to supply one, so that
   * combination could never do anything — the only signal to combine was the
   * client's own timeout.
   */
  describe("the caller's cancellation", () => {
    it("aborts a request that has not been answered yet", async () => {
      const controller = new AbortController();
      installFetch(neverAnswers);

      const pending = api.apiClient.get("jobs", {
        signal: controller.signal,
      });
      await vi.waitFor(() => expect(requests).toHaveLength(1));

      controller.abort();

      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      expect(requests[0].signal.aborted).toBe(true);
    });

    it("reaches the request that is replayed after a refresh", async () => {
      const controller = new AbortController();
      let protectedCalls = 0;
      installFetch((request) => {
        if (request.url.endsWith("/auth/refresh")) return jsonResponse({});
        protectedCalls += 1;
        // The replay is left hanging: a cancellation that only works on a
        // settled request is not a cancellation.
        return protectedCalls === 1
          ? jsonResponse({}, 401)
          : neverAnswers(request);
      });

      const pending = api.apiClient.get("jobs", { signal: controller.signal });
      // Let the 401, the refresh and the replay's dispatch all happen.
      await vi.waitFor(() => expect(protectedCalls).toBe(2));

      controller.abort();

      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      // Cancelled on the wire, not merely ignored by the caller.
      expect(requests.at(-1)?.signal.aborted).toBe(true);
    });

    it("cancels a replay dispatched after the caller had already given up", async () => {
      const controller = new AbortController();
      let refreshCalls = 0;
      let protectedCalls = 0;
      installFetch((request) => {
        if (request.url.endsWith("/auth/refresh")) {
          refreshCalls += 1;
          // Still refreshing when the caller navigates away; the replay is
          // therefore dispatched with a signal that is already aborted.
          return new Promise<Response>((resolve) => {
            controller.signal.addEventListener("abort", () =>
              resolve(jsonResponse({})),
            );
          });
        }
        protectedCalls += 1;
        return protectedCalls === 1
          ? jsonResponse({}, 401)
          : neverAnswers(request);
      });

      const pending = api.apiClient.get("jobs", { signal: controller.signal });
      await vi.waitFor(() => expect(refreshCalls).toBe(1));

      controller.abort();

      await expect(pending).rejects.toMatchObject({ name: "AbortError" });
      expect(protectedCalls).toBe(2);
      expect(requests.at(-1)?.signal.aborted).toBe(true);
    });

    it("still reports a timeout as a timeout, not as a cancellation", async () => {
      installFetch(neverAnswers);

      await expect(
        api.apiClient.postBlob("resumes/r1/export", {}, 20),
      ).rejects.toMatchObject({ code: "TIMEOUT_ERROR" });
    });
  });
});

/**
 * Collapsing every non-HTTP failure into one anonymous NETWORK_ERROR made a
 * timeout, an unreachable server and a bug in this file indistinguishable —
 * and threw away the original error with them.
 */
describe("apiClient failure classification", () => {
  let api: ApiModule;
  let originalFetch: typeof globalThis.fetch;

  beforeEach(async () => {
    originalFetch = globalThis.fetch;
    vi.stubEnv("VITE_API_BASE_URL", BASE);
    vi.resetModules();
    api = await import("../api");
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  function failWith(error: unknown) {
    globalThis.fetch = vi.fn(async () => {
      throw error;
    }) as unknown as typeof globalThis.fetch;
  }

  // POST throughout: ky retries GET on a transport failure, and the backoff
  // would add seconds to every case below without changing what is asserted.
  it("reports a transport failure as NETWORK_ERROR and keeps the cause", async () => {
    const cause = new TypeError("Failed to fetch");
    failWith(cause);

    await expect(api.apiClient.post("jobs", {})).rejects.toMatchObject({
      code: "NETWORK_ERROR",
      message: "",
      cause,
    });
  });

  it("tells a timeout apart from an unreachable server", async () => {
    failWith(new DOMException("The operation timed out.", "TimeoutError"));

    const error = await api.apiClient.post("jobs", {}).catch((e: unknown) => e);

    expect(error).toBeInstanceOf(api.ApiError);
    expect((error as InstanceType<ApiModule["ApiError"]>).code).toBe(
      "TIMEOUT_ERROR",
    );
  });

  it("still hands a caller-cancelled request back as a cancellation", async () => {
    const abort = new DOMException("aborted", "AbortError");
    failWith(abort);

    await expect(api.apiClient.post("jobs", {})).rejects.toBe(abort);
  });

  // A 200 whose body is not JSON is the API or a proxy misbehaving, not the
  // network — and it is worth a log, because nothing else will notice.
  it("reports a malformed success body as INVALID_RESPONSE and logs it", async () => {
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});
    globalThis.fetch = vi.fn(
      async () =>
        new Response("<html>not json</html>", {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
    ) as unknown as typeof globalThis.fetch;

    await expect(
      api.apiClient.get("jobs?q=secret+search"),
    ).rejects.toMatchObject({
      code: "INVALID_RESPONSE",
      message: "",
    });

    expect(logged).toHaveBeenCalledOnce();
    const line = String(logged.mock.calls[0][0]);
    expect(line).toContain("GET jobs");
    // The query string can carry what the customer typed; it never reaches a log.
    expect(line).not.toContain("secret");
  });

  it("surfaces a bug on this side rather than disguising it as a network problem", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    failWith(new RangeError("someone passed a bad option"));

    await expect(api.apiClient.post("jobs", {})).rejects.toMatchObject({
      code: "UNEXPECTED_ERROR",
    });
  });

  it("never puts the transport's own text in the message", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    for (const failure of [
      new TypeError("Failed to fetch http://internal.api:8080/v1/jobs"),
      new DOMException("timed out", "TimeoutError"),
      new RangeError("Bearer eyJhbGciOi"),
    ]) {
      failWith(failure);
      const error = (await api.apiClient
        .post("jobs", {})
        .catch((e: unknown) => e)) as Error;

      expect(error.message).toBe("");
    }
  });
});
