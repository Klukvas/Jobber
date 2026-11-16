import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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

// After another tab's logout the replay is refused instead; see "session
// generation across tabs" below.
it("automatic refresh in another tab waits for the cookie lock, then replays normally", async () => {
  installLocks();
  const { first, second } = await clients();
  let finishRefresh: (value: Response) => void = () => {};
  let protectedCalls = 0;
  const paths: string[] = [];
  vi.stubGlobal("fetch", vi.fn((request: Request) => {
    const path = new URL(request.url).pathname;
    paths.push(path);
    if (path.endsWith("/refresh") && paths.length === 1) return new Promise<Response>((resolve) => { finishRefresh = resolve; });
    if (path.endsWith("/refresh")) return Promise.resolve(response());
    return Promise.resolve(response(++protectedCalls === 1 ? 401 : 200));
  }));
  const refresh = first.post("auth/refresh");
  await vi.waitFor(() => expect(paths).toHaveLength(1));
  const protectedRequest = second.get("jobs");
  await vi.waitFor(() => expect(paths).toHaveLength(2));
  await new Promise((resolve) => setTimeout(resolve, 20));
  expect(paths).toHaveLength(2);
  finishRefresh(response());
  await Promise.all([refresh, protectedRequest]);
  expect(paths).toEqual(["/api/v1/auth/refresh", "/api/v1/jobs", "/api/v1/auth/refresh", "/api/v1/jobs"]);
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

// --- Session generation across tabs -----------------------------------------

const user = (id: string) => ({ id, name: id, email: `${id}@example.com`, locale: "en", created_at: "" });
const credentials = (id: string) => ({ email: `${id}@example.com`, password: "password123" });

/** A broadcast channel whose messages wait until the test delivers them. */
function installChannels() {
  const open = new Set<FakeChannel>();
  const queued: Array<() => void> = [];
  class FakeChannel {
    onmessage: ((event: MessageEvent) => void) | null = null;
    readonly name: string;
    constructor(name: string) {
      this.name = name;
      open.add(this);
    }
    postMessage(data: unknown) {
      open.forEach((other) => {
        if (other !== this && other.name === this.name) {
          queued.push(() => other.onmessage?.(new MessageEvent("message", { data })));
        }
      });
    }
    close() { open.delete(this); }
  }
  vi.stubGlobal("BroadcastChannel", FakeChannel);
  return { deliver: () => queued.splice(0).forEach((send) => send()) };
}

/** One server and one cookie jar for every tab; `held` paths answer on release. */
function cookieServer(held: string[] = []) {
  const state: { jar: string | undefined } = { jar: "a" };
  const calls: string[] = [];
  const holding = new Set(held);
  const gates = new Map<string, (status: number) => void>();
  const fetch = vi.fn(async (request: Request) => {
    const path = new URL(request.url).pathname.replace("/api/v1/", "");
    const body = await request.clone().text();
    const email = path === "auth/login" ? (JSON.parse(body) as { email: string }).email : "";
    const id = email.split("@")[0];
    calls.push(path === "auth/logout" ? "logout"
      : path === "auth/login" ? `login ${id}`
        : path === "auth/refresh" ? `refresh as ${state.jar}`
          : `${request.method} ${path} as ${state.jar}${body ? ` ${body}` : ""}`);
    let status = path === "auth/refresh" && !state.jar ? 401 : 200;
    if (holding.delete(path)) {
      status = await new Promise<number>((resolve, reject) => {
        gates.set(path, resolve);
        request.signal.addEventListener("abort", () => reject(request.signal.reason), { once: true });
      });
    }
    if (status === 200 && path === "auth/logout") state.jar = undefined;
    if (status === 200 && path === "auth/login") state.jar = id;
    return path === "auth/login" && status === 200
      ? new Response(JSON.stringify({ user: user(id) }), { headers: { "content-type": "application/json" } })
      : response(status);
  });
  vi.stubGlobal("fetch", fetch);
  return {
    calls,
    jar: () => state.jar,
    async release(path: string, status = 200) {
      await vi.waitFor(() => expect(gates.has(path)).toBe(true));
      gates.get(path)?.(status);
    },
  };
}

/** A tab: its own module instances, sharing storage, locks and cookies. */
async function openTab(signedInAs?: string) {
  vi.resetModules();
  const { apiClient } = await import("../api");
  const { authService } = await import("../authService");
  const { useAuthStore } = await import("@/stores/authStore");
  const queries = (await import("@/shared/lib/queryClient")).getQueryClient();
  if (signedInAs) {
    useAuthStore.getState().setAuth(user(signedInAs));
    queries.setQueryData(["jobs"], [`${signedInAs}'s job`]);
  }
  const signedIn = () => useAuthStore.getState().user?.id;
  const signIn = async (id: string) => {
    const { user: account } = await authService.login(credentials(id));
    useAuthStore.getState().setAuth(account);
  };
  return { api: apiClient, auth: authService, queries, signedIn, signIn };
}

const persistedUser = () => {
  const entry = localStorage.getItem("jobber-auth");
  return entry ? (JSON.parse(entry) as { state: { user: { id: string } | null } }).state.user?.id : undefined;
};

describe("session generation across tabs", () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("a repeated logout from a tab that is behind leaves the next sign-in's cookies alone", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const three = await openTab("a");
    const server = cookieServer(["auth/logout"]);

    const logout = one.auth.logout();
    await vi.waitFor(() => expect(server.calls).toEqual(["logout"]));
    const login = two.signIn("b");
    const repeated = three.auth.logout().catch((error: unknown) => error);
    await new Promise((resolve) => setTimeout(resolve, 20));
    await server.release("auth/logout");
    await Promise.all([logout, login]);

    expect(await repeated).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(["logout", "login b"]);
    expect(server.jar()).toBe("b");
    expect(two.signedIn()).toBe("b");
    expect(three.signedIn()).toBeUndefined();
    expect(three.queries.getQueryData(["jobs"])).toBeUndefined();
    expect(persistedUser()).toBe("b");
  });

  it("a 401 for a session another tab logged out is neither refreshed nor replayed as the next account", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const three = await openTab("a");
    const server = cookieServer(["jobs", "auth/logout"]);
    vi.spyOn(console, "error").mockImplementation(() => {});

    let settled = false;
    const create = three.api.post("jobs", { title: "A's job" })
      .catch((error: unknown) => error)
      .finally(() => { settled = true; });
    await vi.waitFor(() => expect(server.calls).toHaveLength(1));
    const logout = one.auth.logout();
    await vi.waitFor(() => expect(server.calls).toHaveLength(2));
    const login = two.signIn("b");
    await server.release("jobs", 401);
    await new Promise((resolve) => setTimeout(resolve, 20));
    // Refused at once, not after queueing behind the logout for the lock.
    expect(settled).toBe(true);
    await server.release("auth/logout");
    await Promise.all([logout, login, create]);

    expect(await create).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(['POST jobs as a {"title":"A\'s job"}', "logout", "login b"]);
    expect(server.jar()).toBe("b");
    expect(two.signedIn()).toBe("b");
    expect(three.signedIn()).toBeUndefined();
    expect(three.queries.getQueryData(["jobs"])).toBeUndefined();
  });

  it("a refresh queued behind another tab's sign-in is dropped once it gets the lock", async () => {
    installLocks();
    installChannels();
    const two = await openTab();
    const three = await openTab("a");
    const server = cookieServer(["auth/login", "jobs"]);

    const login = two.signIn("b");
    await vi.waitFor(() => expect(server.calls).toEqual(["login b"]));
    const read = three.api.get("jobs").catch((error: unknown) => error);
    // The read is refused; its refresh then queues behind the login.
    await server.release("jobs", 401);
    await new Promise((resolve) => setTimeout(resolve, 20));
    await server.release("auth/login");
    await Promise.all([login, read]);

    expect(await read).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(["login b", "GET jobs as a"]);
    expect(two.signedIn()).toBe("b");
    expect(three.signedIn()).toBeUndefined();
  });

  it("a broadcast signs an idle tab out and cancels its requests without erasing the new account", async () => {
    installLocks();
    const channels = installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const server = cookieServer(["jobs"]);

    const read = two.api.get("jobs").catch((error: unknown) => error);
    await vi.waitFor(() => expect(server.calls).toHaveLength(1));
    await one.auth.logout();
    await one.signIn("b");
    expect(two.signedIn()).toBe("a");

    channels.deliver();

    expect(two.signedIn()).toBeUndefined();
    expect(two.queries.getQueryData(["jobs"])).toBeUndefined();
    expect(await read).toMatchObject({ name: "AbortError" });
    expect(one.signedIn()).toBe("b");
    expect(persistedUser()).toBe("b");
  });

  it("keeps single-tab logout, login and refresh as before", async () => {
    installLocks();
    installChannels();
    const tab = await openTab("a");
    const server = cookieServer();

    await tab.auth.logout();
    await tab.signIn("b");
    await tab.auth.refresh({ refresh_token: "" });

    expect(server.calls).toEqual(["logout", "login b", "refresh as b"]);
    expect(tab.signedIn()).toBe("b");
    expect(persistedUser()).toBe("b");
  });

  it("falls back to the channel when storage is blocked", async () => {
    installLocks();
    const channels = installChannels();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(globalThis, "localStorage", "get").mockImplementation(() => {
      throw new DOMException("Storage is blocked", "SecurityError");
    });
    const one = await openTab("a");
    const two = await openTab("a");
    const server = cookieServer();

    await one.auth.logout();
    channels.deliver();
    expect(two.signedIn()).toBeUndefined();

    await two.signIn("b");
    expect(server.calls).toEqual(["logout", "login b"]);
    expect(two.signedIn()).toBe("b");
  });

  it("keeps logout and refresh working when storage reads but cannot be written", async () => {
    installLocks();
    installChannels();
    const write = Storage.prototype.setItem;
    // Only the generation write fails; the auth store's own writes are a separate concern.
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(function (this: Storage, key: string, value: string) {
      if (key === "cursopus-session-generation") throw new DOMException("Quota exceeded", "QuotaExceededError");
      write.call(this, key, value);
    });
    const tab = await openTab("a");
    const server = cookieServer(["jobs"]);

    await tab.auth.logout();
    await tab.signIn("b");
    const read = tab.api.get("jobs");
    await server.release("jobs", 401);
    await read;

    expect(server.calls).toEqual(["logout", "login b", "GET jobs as b", "refresh as b", "GET jobs as b"]);
    expect(tab.signedIn()).toBe("b");
  });

  it("does not replay when another tab logs out while this tab's refresh is in flight", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const three = await openTab("a");
    const server = cookieServer(["jobs", "auth/refresh"]);

    const create = three.api.post("jobs", { title: "A's job" }).catch((error: unknown) => error);
    await server.release("jobs", 401);
    await vi.waitFor(() => expect(server.calls).toHaveLength(2));
    const logout = one.auth.logout();
    await server.release("auth/refresh");
    await Promise.all([logout, create]);

    expect(await create).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(['POST jobs as a {"title":"A\'s job"}', "refresh as a", "logout"]);
    expect(three.signedIn()).toBeUndefined();
  });

  it("does not report a failed refresh as this session's end once another tab has moved on", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const three = await openTab("a");
    const server = cookieServer(["jobs", "auth/refresh"]);
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});

    const read = three.api.get("jobs").catch((error: unknown) => error);
    await server.release("jobs", 401);
    await vi.waitFor(() => expect(server.calls).toHaveLength(2));
    const logout = one.auth.logout();
    await server.release("auth/refresh", 401);
    await Promise.all([logout, read]);

    expect(await read).toMatchObject({ name: "AbortError" });
    expect(logged).not.toHaveBeenCalled();
    expect(server.calls).toEqual(["GET jobs as a", "refresh as a", "logout"]);
  });

  it("drops a login whose response arrives after another tab started logging out", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const two = await openTab();
    const server = cookieServer(["auth/login"]);

    const login = two.signIn("b").catch((error: unknown) => error);
    await vi.waitFor(() => expect(server.calls).toEqual(["login b"]));
    const logout = one.auth.logout();
    await server.release("auth/login");
    await Promise.all([logout, login]);

    expect(await login).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(["login b", "logout"]);
    expect(server.jar()).toBeUndefined();
    expect(two.signedIn()).toBeUndefined();
  });

  it("does not send a login that queued behind another tab's sign-in", async () => {
    installLocks();
    installChannels();
    const two = await openTab();
    const four = await openTab();
    const server = cookieServer(["auth/login"]);

    const first = two.signIn("b");
    await vi.waitFor(() => expect(server.calls).toEqual(["login b"]));
    const second = four.signIn("c").catch((error: unknown) => error);
    await new Promise((resolve) => setTimeout(resolve, 20));
    await server.release("auth/login");
    await Promise.all([first, second]);

    expect(await second).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(["login b"]);
    expect(server.jar()).toBe("b");
    expect(two.signedIn()).toBe("b");
  });
  // A session that expired in one tab is renewed by signing in again there.
  // The same person's other tabs keep their state, and their own requests may
  // refresh and replay: the cookies are still that account's.
  it.each([
    ["after the broadcast", true],
    ["before it arrives", false],
  ])("keeps another tab of the same account signed in when it signs in again (%s)", async (_label, delivered) => {
    installLocks();
    const channels = installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const server = cookieServer(["jobs"]);

    await one.signIn("a");
    if (delivered) channels.deliver();
    const read = two.api.get("jobs");
    await server.release("jobs", 401);
    await read;

    expect(two.signedIn()).toBe("a");
    expect(two.queries.getQueryData(["jobs"])).toEqual(["a's job"]);
    expect(server.calls).toEqual(["login a", "GET jobs as a", "refresh as a", "GET jobs as a"]);
    expect(persistedUser()).toBe("a");
  });

  it("still signs another tab out when a different account signs in", async () => {
    installLocks();
    const channels = installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const server = cookieServer(["jobs"]);

    await one.signIn("b");
    const read = two.api.get("jobs").catch((error: unknown) => error);
    await server.release("jobs", 401);
    channels.deliver();

    expect(await read).toMatchObject({ name: "AbortError" });
    expect(server.calls).toEqual(["login b", "GET jobs as b"]);
    expect(two.signedIn()).toBeUndefined();
    expect(persistedUser()).toBe("b");
  });
  it("a tab that missed the same account signing in again still logs out on the server", async () => {
    installLocks();
    installChannels();
    const one = await openTab("a");
    const two = await openTab("a");
    const server = cookieServer();

    await one.signIn("a");
    await two.auth.logout();

    expect(server.calls).toEqual(["login a", "logout"]);
    expect(server.jar()).toBeUndefined();
    expect(two.signedIn()).toBeUndefined();
  });
});
