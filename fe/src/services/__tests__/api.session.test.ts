import { afterEach, beforeEach, expect, it, vi } from "vitest";
vi.mock("@/shared/lib/aiCredits", () => ({ confirmAIAction: vi.fn(async () => undefined) }));

const user = (id: string) => ({ id, name: id, email: `${id}@example.com`, locale: "en", created_at: "" });
const response = (status = 200) => new Response(JSON.stringify({ error_code: "UNAUTHORIZED" }), { status, headers: { "content-type": "application/json" } });
beforeEach(() => { vi.resetModules(); vi.stubEnv("VITE_API_BASE_URL", "http://api.test/api/v1"); });
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); vi.unstubAllEnvs(); vi.restoreAllMocks(); });

it.each(["logout", "switch"])("ignores an old 401 after %s without refresh, replay or clearing the new user", async (action) => {
  const { useAuthStore } = await import("@/stores/authStore");
  useAuthStore.getState().setAuth(user("a"));
  const { apiClient } = await import("../api");
  let answer: (value: Response) => void = () => {};
  const fetch = vi.fn(() => new Promise<Response>((resolve) => { answer = resolve; }));
  vi.stubGlobal("fetch", fetch);
  const pending = apiClient.get("subscription").catch((error: unknown) => error);
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  if (action === "logout") useAuthStore.getState().clearAuth();
  else useAuthStore.getState().setAuth(user("b"));
  answer(response(401));
  expect(await pending).toMatchObject({ name: "AbortError" });
  expect(fetch).toHaveBeenCalledOnce();
  expect(useAuthStore.getState().user?.id).toBe(action === "logout" ? undefined : "b");
});

it("cancels a refresh in flight and does not replay into a new session", async () => {
  const { useAuthStore } = await import("@/stores/authStore");
  useAuthStore.getState().setAuth(user("a"));
  const { apiClient } = await import("../api");
  let finishRefresh: (value: Response) => void = () => {};
  const fetch = vi.fn((request: Request) => request.url.endsWith("/auth/refresh")
    ? new Promise<Response>((resolve) => { finishRefresh = resolve; }) : Promise.resolve(response(401)));
  vi.stubGlobal("fetch", fetch);
  const logged = vi.spyOn(console, "error").mockImplementation(() => {});
  const pending = apiClient.get("subscription").catch((error: unknown) => error);
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
  useAuthStore.getState().setAuth(user("b"));
  finishRefresh(response());
  expect(await pending).toMatchObject({ name: "AbortError" });
  expect(fetch).toHaveBeenCalledTimes(2);
  expect(logged).not.toHaveBeenCalled();
  expect(useAuthStore.getState().user?.id).toBe("b");
});

it("keeps normal refresh working and preserves requests when the same profile updates", async () => {
  const { useAuthStore } = await import("@/stores/authStore");
  useAuthStore.getState().setAuth(user("a"));
  const { apiClient } = await import("../api");
  let calls = 0;
  vi.stubGlobal("fetch", vi.fn((request: Request) => {
    if (request.url.endsWith("/auth/refresh")) {
      useAuthStore.getState().setAuth({ ...user("a"), name: "Updated" });
      return Promise.resolve(response());
    }
    return Promise.resolve(response(++calls === 1 ? 401 : 200));
  }));
  await expect(apiClient.get("subscription")).resolves.toEqual({ error_code: "UNAUTHORIZED" });
  expect(calls).toBe(2);
});

it("logout ends local requests before waiting for the server", async () => {
  const { useAuthStore } = await import("@/stores/authStore");
  useAuthStore.getState().setAuth(user("a"));
  const { authService } = await import("../authService");
  const fetch = vi.fn(async () => response());
  vi.stubGlobal("fetch", fetch);
  const pending = authService.logout();
  expect(useAuthStore.getState().isAuthenticated).toBe(false);
  await pending;
  expect(fetch).toHaveBeenCalledOnce();
});

it.each(["login", "register", "refresh"] as const)("waits for logout cookies before sending %s", async (action) => {
  const { useAuthStore } = await import("@/stores/authStore");
  const { sessionSignal } = await import("@/shared/lib/sessionLifetime");
  useAuthStore.getState().setAuth(user("a"));
  const oldSession = sessionSignal();
  const { authService } = await import("../authService");
  let cookieOwner: string | undefined = "a";
  let finishLogout = () => {};
  const fetch = vi.fn((request: Request) => {
    if (request.url.endsWith("/auth/logout")) {
      return new Promise<Response>((resolve) => {
        finishLogout = () => { cookieOwner = undefined; resolve(response()); };
      });
    }
    cookieOwner = "b";
    return Promise.resolve(new Response(JSON.stringify({ user: user("b") }), {
      headers: { "content-type": "application/json" },
    }));
  });
  vi.stubGlobal("fetch", fetch);
  const logout = authService.logout();
  expect(oldSession.aborted).toBe(true);
  expect(useAuthStore.getState().isAuthenticated).toBe(false);
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  const signIn = action === "refresh"
    ? authService.refresh({ refresh_token: "b" })
    : authService[action]({ email: "b@example.com", password: "password123" });
  // Allow the real ky request pipeline to dispatch if there is no barrier.
  await new Promise((resolve) => setTimeout(resolve, 20));
  const callsBeforeLogoutFinished = fetch.mock.calls.length;
  finishLogout();
  await logout;
  await signIn;
  expect(callsBeforeLogoutFinished).toBe(1);
  expect(cookieOwner).toBe("b");
  expect(fetch.mock.calls.map(([request]) => new URL(request.url).pathname))
    .toEqual(["/api/v1/auth/logout", `/api/v1/auth/${action}`]);
});

it("releases a waiting login after a failed logout and allows another logout", async () => {
  const { authService } = await import("../authService");
  let failLogout: (error: Error) => void = () => {};
  const fetch = vi.fn((request: Request) => request.url.endsWith("/auth/logout")
    ? new Promise<Response>((_resolve, reject) => { failLogout = reject; })
    : Promise.resolve(response()));
  vi.stubGlobal("fetch", fetch);
  const logout = authService.logout().catch((error: unknown) => error);
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  const login = authService.login({ email: "b@example.com", password: "password123" });
  failLogout(new TypeError("Failed to fetch"));
  expect(await logout).toMatchObject({ code: "NETWORK_ERROR" });
  await login;
  fetch.mockResolvedValue(response());
  await authService.logout();
  expect(fetch).toHaveBeenCalledTimes(3);
});

it("shares duplicate logouts so no second late response can clear the next session", async () => {
  const { authService } = await import("../authService");
  let finishLogout: (response: Response) => void = () => {};
  const fetch = vi.fn((request: Request) => request.url.endsWith("/auth/logout")
    ? new Promise<Response>((resolve) => { finishLogout = resolve; })
    : Promise.resolve(response()));
  vi.stubGlobal("fetch", fetch);
  const first = authService.logout();
  const second = authService.logout();
  const login = authService.login({ email: "b@example.com", password: "password123" });
  await vi.waitFor(() => expect(fetch).toHaveBeenCalledOnce());
  finishLogout(response());
  await Promise.all([first, second, login]);
  expect(fetch).toHaveBeenCalledTimes(2);
});

it("aborts a timed-out logout transport before sending the waiting login", async () => {
  vi.useFakeTimers();
  const { authService } = await import("../authService");
  const events: string[] = [];
  const fetch = vi.fn((request: Request) => {
    if (request.url.endsWith("/auth/logout")) {
      events.push("logout");
      return new Promise<Response>((_resolve, reject) => {
        request.signal.addEventListener("abort", () => {
          events.push("abort logout");
          reject(request.signal.reason);
        }, { once: true });
      });
    }
    events.push("login");
    return Promise.resolve(response());
  });
  vi.stubGlobal("fetch", fetch);
  const logout = authService.logout().catch((error: unknown) => error);
  const login = authService.login({ email: "b@example.com", password: "password123" });
  await vi.waitFor(() => expect(events).toEqual(["logout"]));
  await vi.advanceTimersByTimeAsync(30_000);
  expect(await logout).toMatchObject({ code: "TIMEOUT_ERROR" });
  await login;
  expect(events).toEqual(["logout", "abort logout", "login"]);
});
