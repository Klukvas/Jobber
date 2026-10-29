import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  focusManager,
  onlineManager,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { apiClient, ApiError } from "@/services/api";
import { adminService, type AdminDetail } from "@/services/adminService";
import { useAuthStore } from "@/stores/authStore";
import {
  adminKeys,
  bindAdminClient,
  currentAdminRequest,
} from "@/shared/lib/adminCapability";
import { useAdminAccess } from "@/shared/hooks/useAdminAccess";
import i18n from "@/shared/lib/i18n";
import AdminPage from "../Admin";

vi.mock("@/services/api", async (original) => ({
  ...(await original<typeof import("@/services/api")>()),
  apiClient: { get: vi.fn(), post: vi.fn(), patch: vi.fn() },
}));
const actor = {
  id: "actor-a",
  email: "admin@example.com",
  name: "Operator",
  locale: "en",
  created_at: "2026-10-29",
};
const target = {
  ...actor,
  id: "target",
  name: "Protected name",
  email: "target@example.com",
  email_verified: true,
  provider_plan: "free",
  provider_status: "free",
  manual_until: null,
};
const detail: AdminDetail = {
  user: target,
  effective_plan: "free",
  provider_subscription_id: null,
  provider_account_id: null,
  provider_cancel_at: null,
  provider_period_end: null,
  manual_grants: [],
};
const list = { users: [target], total: 50, page: 1, limit: 20 };
const forbidden = () => new ApiError("Forbidden", "FORBIDDEN", 403);
function deferred<T>() {
  let resolve: (value: T) => void = () => {
    throw new Error("Not initialized");
  };
  let reject: (error: unknown) => void = () => {
    throw new Error("Not initialized");
  };
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
let client: QueryClient;
let unbind: () => void;
function Nav() {
  const access = useAdminAccess();
  return access.data?.allowed && !access.isError ? (
    <nav aria-label="Admin navigation">Admin link</nav>
  ) : null;
}
function setup() {
  render(
    <QueryClientProvider client={client}>
      <Nav />
      <AdminPage />
    </QueryClientProvider>,
  );
}
beforeEach(async () => {
  vi.resetAllMocks();
  await i18n.changeLanguage("en");
  useAuthStore.setState({ user: null, isAuthenticated: false });
  useAuthStore.setState({ user: actor, isAuthenticated: true });
  client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false },
      mutations: { retry: false },
    },
  });
  unbind = bindAdminClient(client);
  vi.mocked(apiClient.get).mockImplementation(async (url) =>
    url === "admin/access"
      ? { allowed: true }
      : url.includes("?")
        ? list
        : detail,
  );
  vi.mocked(apiClient.patch).mockResolvedValue({ saved: true });
  vi.mocked(apiClient.post).mockResolvedValue({ saved: true });
});
afterEach(() => {
  cleanup();
  unbind();
  client.clear();
  focusManager.setFocused(undefined);
  onlineManager.setOnline(true);
  vi.useRealTimers();
});
async function selectTarget() {
  await userEvent.click(
    await screen.findByRole("button", { name: /Protected name/ }),
  );
  await screen.findByRole("button", { name: "Save profile" });
}
function expectNoProtectedData() {
  expect(
    client
      .getQueriesData({
        predicate: (query) =>
          ["admin-user", "admin-users"].includes(String(query.queryKey[0])),
      })
      .every(([, data]) => data === undefined),
  ).toBe(true);
}

describe("admin capability boundary", () => {
  it.each([401, 403])(
    "removes cached list, details, actions and navigation after final mutation %s",
    async (status) => {
      setup();
      await selectTarget();
      vi.mocked(apiClient.patch).mockRejectedValue(
        new ApiError("Denied", "DENIED", status),
      );
      await userEvent.click(
        screen.getByRole("button", { name: "Save profile" }),
      );
      await screen.findByRole("heading", {
        name: "Administrator access required",
      });
      expect(screen.queryByText("Protected name")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Save profile" }),
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Grant manual Pro access" }),
      ).not.toBeInTheDocument();
      expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
      expectNoProtectedData();
      expect(
        client.getQueryCache().findAll({
          predicate: (query) =>
            ["admin-user", "admin-users"].includes(String(query.queryKey[0])),
        }),
      ).toHaveLength(0);
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    },
  );
  it("denies cached detail and nav after list denial and discards concurrent detail and access successes", async () => {
    setup();
    await selectTarget();
    const pendingDetail = deferred<AdminDetail>();
    const pendingAccess = deferred<{ allowed: boolean }>();
    vi.mocked(apiClient.get).mockImplementation((url) =>
      url === "admin/access"
        ? pendingAccess.promise
        : url.includes("?")
          ? Promise.reject(forbidden())
          : pendingDetail.promise,
    );
    const detailRead = client.refetchQueries({
      queryKey: adminKeys.detail(currentAdminRequest(), target.id),
    });
    const accessRead = client.refetchQueries({ queryKey: ["admin-access"] });
    await userEvent.type(
      screen.getByRole("textbox", { name: "Search users by name or email" }),
      "new",
    );
    await userEvent.click(screen.getByRole("button", { name: "Search" }));
    await screen.findByRole("heading", {
      name: "Administrator access required",
    });
    await act(async () => {
      pendingDetail.resolve(detail);
      pendingAccess.resolve({ allowed: true });
      await Promise.all([detailRead, accessRead]);
    });
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    expectNoProtectedData();
    // A fresh explicit probe can restore capability after server permission returns.
    vi.mocked(apiClient.get).mockImplementation(async (url) =>
      url === "admin/access" ? { allowed: true } : list,
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Check access again" }),
    );
    await screen.findByRole("button", { name: /Protected name/ });
  });
  it("evicts cached detail on detail refetch denial, including standalone observers", async () => {
    setup();
    await selectTarget();
    vi.mocked(apiClient.get).mockRejectedValue(forbidden());
    await act(async () => {
      await client.refetchQueries({
        queryKey: adminKeys.detail(currentAdminRequest(), target.id),
      });
    });
    await screen.findByRole("heading", {
      name: "Administrator access required",
    });
    expect(
      screen.queryByRole("button", { name: "Save profile" }),
    ).not.toBeInTheDocument();
    expectNoProtectedData();
  });
  it.each([
    ["success", "other actor"],
    ["denial", "other actor"],
    ["success", "same actor relogin"],
    ["denial", "same actor relogin"],
  ])("ignores old actor %s after %s", async (outcome, switchKind) => {
    const pending = deferred<AdminDetail>();
    vi.mocked(apiClient.get).mockReturnValueOnce(pending.promise);
    const oldRequest = adminService
      .detail(target.id)
      .catch((error: unknown) => error);
    const oldScope = currentAdminRequest();
    client.setQueryData(adminKeys.detail(oldScope, target.id), detail);
    if (switchKind === "same actor relogin") {
      useAuthStore.setState({ user: null, isAuthenticated: false });
      useAuthStore.setState({ user: actor, isAuthenticated: true });
    } else useAuthStore.setState({ user: { ...actor, id: "actor-b" } });
    expect(
      client
        .getQueryCache()
        .findAll({ predicate: (query) => query.queryKey[0] === "admin-user" }),
    ).toHaveLength(0);
    await adminService.access();
    if (outcome === "success") pending.resolve(detail);
    else pending.reject(forbidden());
    const result = await oldRequest;
    expect(result).toBeDefined();
    if (outcome === "success")
      expect(result).toHaveProperty("name", "AbortError");
    else expect(result).toHaveProperty("status", 403);
    expect(currentAdminRequest().denied).toBe(false);
    expect(
      client.getQueryData(adminKeys.detail(oldScope, target.id)),
    ).toBeUndefined();
    useAuthStore.setState({ user: actor });
    expect(currentAdminRequest().session).not.toBe(oldScope.session);
    expect(currentAdminRequest().denied).toBe(false);
  });
  it("does not run mutation success invalidations from an old actor", async () => {
    setup();
    await selectTarget();
    const pending = deferred<unknown>();
    vi.mocked(apiClient.patch).mockReturnValue(pending.promise);
    await userEvent.click(screen.getByRole("button", { name: "Save profile" }));
    const invalidate = vi.spyOn(client, "invalidateQueries");
    await act(async () => {
      useAuthStore.setState({ user: { ...actor, id: "actor-b" } });
      pending.resolve({ saved: true });
    });
    await waitFor(() => expect(currentAdminRequest().actor).toBe("actor-b"));
    expect(invalidate).not.toHaveBeenCalled();
  });
  it("revalidates access on focus despite the application default", async () => {
    setup();
    await screen.findByRole("button", { name: /Protected name/ });
    vi.mocked(apiClient.get).mockRejectedValue(forbidden());
    act(() => focusManager.setFocused(false));
    act(() => focusManager.setFocused(true));
    await screen.findByRole("heading", {
      name: "Administrator access required",
    });
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  });
  it("revalidates within 30 seconds and treats allowed:false as denial", async () => {
    vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] });
    setup();
    await screen.findByRole("button", { name: /Protected name/ });
    vi.mocked(apiClient.get).mockResolvedValue({ allowed: false });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(30_100);
    });
    expect(
      screen.getByRole("heading", { name: "Administrator access required" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    expectNoProtectedData();
  });
  it.each(["other actor", "same actor relogin"])(
    "does not dispatch a paused old-form mutation after %s",
    async (switchKind) => {
      setup();
      await selectTarget();
      onlineManager.setOnline(false);
      await userEvent.click(
        screen.getByRole("button", { name: "Save profile" }),
      );
      await waitFor(() =>
        expect(client.getMutationCache().getAll()[0]?.state.isPaused).toBe(
          true,
        ),
      );
      expect(apiClient.patch).not.toHaveBeenCalled();
      await act(async () => {
        if (switchKind === "same actor relogin") {
          useAuthStore.setState({ user: null, isAuthenticated: false });
          useAuthStore.setState({ user: actor, isAuthenticated: true });
        } else useAuthStore.setState({ user: { ...actor, id: "actor-b" } });
        onlineManager.setOnline(true);
      });
      await waitFor(() =>
        expect(client.getMutationCache().getAll()[0]?.state.status).toBe(
          "error",
        ),
      );
      expect(apiClient.patch).not.toHaveBeenCalled();
      expect(apiClient.post).not.toHaveBeenCalled();
    },
  );
  it.each(["success", "denial"])(
    "never paints or caches an old actor list %s after B becomes authorized",
    async (outcome) => {
      const oldList = deferred<typeof list>();
      let firstList = true;
      const nextList = {
        ...list,
        users: [{ ...target, name: "Actor B result" }],
      };
      vi.mocked(apiClient.get).mockImplementation(async (url) => {
        if (url === "admin/access") return { allowed: true };
        if (firstList) {
          firstList = false;
          return oldList.promise;
        }
        return nextList;
      });
      setup();
      await waitFor(() => expect(firstList).toBe(false));
      const oldScope = currentAdminRequest();
      await act(async () => {
        useAuthStore.setState({ user: { ...actor, id: "actor-b" } });
      });
      await screen.findByRole("button", { name: /Actor B result/ });
      await act(async () => {
        if (outcome === "success") oldList.resolve(list);
        else oldList.reject(forbidden());
      });
      expect(
        screen.getByRole("button", { name: /Actor B result/ }),
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: /Protected name/ }),
      ).not.toBeInTheDocument();
      expect(screen.getByRole("navigation")).toBeInTheDocument();
      expect(
        client.getQueryCache().findAll({ queryKey: adminKeys.users(oldScope) }),
      ).toHaveLength(0);
    },
  );
});
