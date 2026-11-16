import { afterEach, beforeEach, expect, it, vi } from "vitest";

vi.mock("@/shared/lib/aiCredits", () => ({ confirmAIAction: vi.fn(async () => undefined) }));

const response = (status = 200) => new Response(JSON.stringify({ ok: true }), {
  status, headers: { "content-type": "application/json" },
});

// One origin-wide queue shared by separately loaded clients, like browser tabs.
function installLocks() {
  let tail = Promise.resolve();
  const request = vi.fn(<T>(name: string, options: LockOptions, callback: () => Promise<T>) => {
    expect(name).toBe("cursopus-auth-cookies");
    const previous = tail;
    let release = () => {};
    tail = new Promise<void>((resolve) => { release = resolve; });
    return new Promise<T>((resolve, reject) => {
      const abort = () => reject(options.signal?.reason);
      options.signal?.addEventListener("abort", abort, { once: true });
      if (options.signal?.aborted) abort();
      void previous.then(async () => {
        options.signal?.removeEventListener("abort", abort);
        try {
          options.signal?.throwIfAborted();
          resolve(await callback());
        } catch (error) {
          reject(error);
        } finally {
          release();
        }
      });
    });
  });
  vi.stubGlobal("navigator", { locks: { request } });
  return request;
}

async function clients() {
  const first = (await import("../api")).apiClient;
  vi.resetModules();
  const second = (await import("../api")).apiClient;
  return { first, second };
}

beforeEach(() => {
  vi.resetModules();
  vi.stubEnv("VITE_API_BASE_URL", "http://api.test/api/v1");
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});

it.each(["login", "register", "refresh"])("waits across tabs for logout cookies before %s", async (action) => {
  installLocks();
  const { first, second } = await clients();
  let cookieOwner: string | undefined = "a";
  let finishLogout = () => {};
  const fetch = vi.fn((request: Request) => {
    if (request.url.endsWith("/auth/logout")) {
      return new Promise<Response>((resolve) => {
        finishLogout = () => { cookieOwner = undefined; resolve(response()); };
      });
    }
    cookieOwner = "b";
    return Promise.resolve(response());
  });
  vi.stubGlobal("fetch", fetch);
  const logout = first.post("auth/logout");
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  const signIn = second.post(`auth/${action}`, {});
  await new Promise((resolve) => setTimeout(resolve, 20));
  const callsBeforeLogout = fetch.mock.calls.length;
  finishLogout();
  await Promise.all([logout, signIn]);
  expect(callsBeforeLogout).toBe(1);
  expect(cookieOwner).toBe("b");
});

it("automatic refresh in another tab waits for logout, then replays normally", async () => {
  installLocks();
  const { first, second } = await clients();
  let finishLogout: (value: Response) => void = () => {};
  let protectedCalls = 0;
  const paths: string[] = [];
  vi.stubGlobal("fetch", vi.fn((request: Request) => {
    const path = new URL(request.url).pathname;
    paths.push(path);
    if (path.endsWith("/logout")) return new Promise<Response>((resolve) => { finishLogout = resolve; });
    if (path.endsWith("/refresh")) return Promise.resolve(response());
    return Promise.resolve(response(++protectedCalls === 1 ? 401 : 200));
  }));
  const logout = first.post("auth/logout");
  await vi.waitFor(() => expect(paths).toHaveLength(1));
  const protectedRequest = second.get("jobs");
  await vi.waitFor(() => expect(paths).toHaveLength(2));
  await new Promise((resolve) => setTimeout(resolve, 20));
  expect(paths.some((path) => path.endsWith("/refresh"))).toBe(false);
  finishLogout(response());
  await Promise.all([logout, protectedRequest]);
  expect(paths).toEqual(["/api/v1/auth/logout", "/api/v1/jobs", "/api/v1/auth/refresh", "/api/v1/jobs"]);
});

it.each(["failure", "timeout"])("releases other tabs after logout %s and coalesces repeated logout", async (outcome) => {
  vi.useFakeTimers();
  installLocks();
  const { first, second } = await clients();
  let failLogout: (error: Error) => void = () => {};
  const events: string[] = [];
  vi.stubGlobal("fetch", vi.fn((request: Request) => {
    if (!request.url.endsWith("/logout")) {
      events.push("login");
      return Promise.resolve(response());
    }
    events.push("logout");
    return new Promise<Response>((_resolve, reject) => {
      failLogout = reject;
      request.signal.addEventListener("abort", () => {
        events.push("abort logout");
        reject(request.signal.reason);
      }, { once: true });
    });
  }));
  const logout = first.post("auth/logout").catch((error: unknown) => error);
  const duplicate = first.post("auth/logout").catch((error: unknown) => error);
  await vi.waitFor(() => expect(events).toEqual(["logout"]));
  const login = second.post("auth/login");
  if (outcome === "failure") failLogout(new TypeError("offline"));
  else await vi.advanceTimersByTimeAsync(30_000);
  expect(await logout).toMatchObject({ code: outcome === "failure" ? "NETWORK_ERROR" : "TIMEOUT_ERROR" });
  await Promise.all([duplicate, login]);
  expect(events).toEqual(outcome === "failure" ? ["logout", "login"] : ["logout", "abort logout", "login"]);
});

it("times out a queued auth operation without sending it outside the lock", async () => {
  vi.useFakeTimers();
  const request = installLocks();
  let release = () => {};
  const holder = request("cursopus-auth-cookies", {}, () => new Promise<void>((resolve) => { release = resolve; }));
  const { second } = await clients();
  const fetch = vi.fn(async () => response());
  vi.stubGlobal("fetch", fetch);
  const login = second.post("auth/login").catch((error: unknown) => error);
  await vi.advanceTimersByTimeAsync(60_000);
  expect(await login).toMatchObject({ code: "TIMEOUT_ERROR" });
  expect(fetch).not.toHaveBeenCalled();
  release();
  await holder;
  await second.post("auth/refresh");
  expect(fetch).toHaveBeenCalledOnce();
});

it.each([200, 503])("releases the logout lock when a %s response body never finishes", async (status) => {
  vi.useFakeTimers();
  installLocks();
  const { first, second } = await clients();
  let bodyController: ReadableStreamDefaultController<Uint8Array> | undefined;
  let logoutSignal: AbortSignal | undefined;
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      bodyController = controller;
      controller.enqueue(new TextEncoder().encode('{"ok":'));
    },
  });
  const fetch = vi.fn((request: Request) => {
    if (!request.url.endsWith("/logout")) return Promise.resolve(response());
    logoutSignal = request.signal;
    request.signal.addEventListener("abort", () => bodyController?.error(request.signal.reason), { once: true });
    return Promise.resolve(new Response(body, { status, headers: { "content-type": "application/json" } }));
  });
  vi.stubGlobal("fetch", fetch);
  let settled = false;
  const logout = first.post("auth/logout").catch((error: unknown) => error).finally(() => { settled = true; });
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  const login = second.post("auth/login");
  try {
    await vi.advanceTimersByTimeAsync(30_000);
    expect(logoutSignal?.aborted).toBe(true);
    expect(settled).toBe(true);
    expect(await logout).toMatchObject({ code: "TIMEOUT_ERROR" });
    await login;
    expect(fetch).toHaveBeenCalledTimes(2);
  } finally {
    // Complete the fixture even on the unfixed client, so no pending stream leaks.
    if (!logoutSignal?.aborted) bodyController?.close();
    await Promise.all([logout, login]);
  }
});

it("cancels a waiting auth request without dispatching it later", async () => {
  const request = installLocks();
  let release = () => {};
  const holder = request("cursopus-auth-cookies", {}, () => new Promise<void>((resolve) => { release = resolve; }));
  const { second } = await clients();
  const fetch = vi.fn(async () => response());
  vi.stubGlobal("fetch", fetch);
  const controller = new AbortController();
  const login = second.post("auth/login", {}, { signal: controller.signal }).catch((error: unknown) => error);
  controller.abort();
  expect(await login).toMatchObject({ name: "AbortError" });
  release();
  await holder;
  await second.post("auth/refresh");
  expect(fetch).toHaveBeenCalledOnce();
});

it("aborts a stalled automatic refresh before releasing the lock to another tab", async () => {
  vi.useFakeTimers();
  installLocks();
  const { first, second } = await clients();
  vi.spyOn(console, "error").mockImplementation(() => {});
  const events: string[] = [];
  vi.stubGlobal("fetch", vi.fn((request: Request) => {
    if (request.url.endsWith("/refresh")) {
      events.push("refresh");
      return new Promise<Response>((_resolve, reject) => {
        request.signal.addEventListener("abort", () => {
          events.push("abort refresh");
          reject(request.signal.reason);
        }, { once: true });
      });
    }
    if (request.url.endsWith("/login")) {
      events.push("login");
      return Promise.resolve(response());
    }
    return Promise.resolve(response(401));
  }));
  const protectedRequest = first.get("jobs", {}).catch((error: unknown) => error);
  await vi.waitFor(() => expect(events).toEqual(["refresh"]));
  const login = second.post("auth/login");
  await vi.advanceTimersByTimeAsync(30_000);
  await Promise.all([protectedRequest, login]);
  expect(events).toEqual(["refresh", "abort refresh", "login"]);
});
