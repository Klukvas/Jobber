import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import i18n from "@/shared/lib/i18n";
import { useAuthStore } from "@/stores/authStore";
import { adminService, type AdminDetail } from "@/services/adminService";
import AdminPage from "../Admin";
import { ApiError } from "@/services/api";
import { AdminUserDetail } from "@/features/admin/AdminUserDetail";

vi.mock("@/services/adminService", () => ({
  adminService: {
    access: vi.fn(),
    stats: vi.fn(),
    setTest: vi.fn(),
    users: vi.fn(),
    detail: vi.fn(),
    update: vi.fn(),
    grant: vi.fn(),
    revoke: vi.fn(),
  },
}));
const user = {
  id: "00000000-0000-4000-8000-000000000001",
  email: "admin@example.com",
  name: "Admin",
  locale: "en",
  created_at: "2026-10-29T00:00:00Z",
  email_verified: true,
  is_test: false,
  provider_plan: "free",
  provider_status: "free",
  manual_until: null,
};
const detail: AdminDetail = {
  user,
  provider_subscription_id: null,
  provider_account_id: null,
  provider_period_end: null,
  provider_cancel_at: null,
  effective_plan: "free",
  manual_grants: [],
};
function setup(node = <AdminPage />) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(<QueryClientProvider client={client}>{node}</QueryClientProvider>);
  return client;
}
beforeEach(async () => {
  vi.resetAllMocks();
  await i18n.changeLanguage("en");
  useAuthStore.setState({ user, isAuthenticated: true });
  vi.mocked(adminService.stats).mockResolvedValue({
    total_users: 21,
    registrations_7_days: 7,
    registrations_30_days: 20,
    daily_registrations: Array.from({ length: 30 }, (_, i) => ({
      date: `2026-10-${String(i + 1).padStart(2, "0")}`,
      count: i % 3,
    })),
    days: 30,
    include_test: false,
    timezone: "Europe/Kyiv",
    as_of: "2026-10-29T10:00:00Z",
  });
  vi.mocked(adminService.setTest).mockResolvedValue({
    saved: true,
    is_test: true,
  });
  vi.mocked(adminService.access).mockResolvedValue({ allowed: true });
  vi.mocked(adminService.users).mockResolvedValue({
    users: [user],
    total: 21,
    page: 1,
    limit: 20,
  });
  vi.mocked(adminService.detail).mockResolvedValue(detail);
  vi.mocked(adminService.update).mockResolvedValue({ saved: true });
  vi.mocked(adminService.grant).mockResolvedValue({ saved: true });
  vi.mocked(adminService.revoke).mockResolvedValue({ saved: true });
});
afterEach(cleanup);

describe("Registration overview and test accounts", () => {
  it("refreshes the current metrics once while keeping the dated snapshot visible", async () => {
    setup();
    const refresh = await screen.findByRole("button", {
      name: "Refresh statistics",
    });
    expect(screen.getByText(/Updated at/)).toHaveTextContent("Oct 29, 2026");
    await userEvent.click(screen.getByRole("button", { name: "7 days" }));
    await screen.findByRole("button", { name: "Refresh statistics" });
    await userEvent.click(
      screen.getByRole("checkbox", {
        name: "Include test accounts in statistics",
      }),
    );
    await screen.findByRole("button", { name: "Refresh statistics" });
    const previous = await vi.mocked(adminService.stats).mock.results.at(-1)
      ?.value;
    let release: ((value: typeof previous) => void) | undefined;
    vi.mocked(adminService.stats).mockReturnValueOnce(
      new Promise((resolve) => {
        release = resolve;
      }),
    );
    const calls = vi.mocked(adminService.stats).mock.calls.length;
    fireEvent.click(refresh);
    fireEvent.click(refresh);
    await waitFor(() =>
      expect(adminService.stats).toHaveBeenCalledTimes(calls + 1),
    );
    expect(adminService.stats).toHaveBeenLastCalledWith(
      7,
      true,
      expect.any(AbortSignal),
    );
    expect(screen.getByRole("button", { name: "Refreshing…" })).toBeDisabled();
    expect(screen.getByText(/Updated at/)).toHaveTextContent("Oct 29, 2026");
    release?.({ ...previous, as_of: "2026-10-30T10:00:00Z" });
    await waitFor(() =>
      expect(screen.getByText(/Updated at/)).toHaveTextContent("Oct 30, 2026"),
    );
    expect(
      screen.getByRole("button", { name: "Refresh statistics" }),
    ).toBeEnabled();
  });
  it("keeps fixed cards, switches chart period and scopes the test checkbox to metrics", async () => {
    setup();
    await screen.findByRole("heading", { name: "Registration overview" });
    await screen.findByRole("img", { name: /Registrations by day/ });
    expect(
      screen.getByRole("heading", { name: "Total accounts" }).parentElement
        ?.parentElement,
    ).toHaveTextContent("21");
    await userEvent.click(screen.getByRole("button", { name: "7 days" }));
    await waitFor(() =>
      expect(adminService.stats).toHaveBeenLastCalledWith(
        7,
        false,
        expect.any(AbortSignal),
      ),
    );
    await userEvent.click(
      screen.getByRole("checkbox", {
        name: "Include test accounts in statistics",
      }),
    );
    await waitFor(() =>
      expect(adminService.stats).toHaveBeenLastCalledWith(
        7,
        true,
        expect.any(AbortSignal),
      ),
    );
    expect(adminService.users).toHaveBeenCalledTimes(1);
    expect(
      screen.getByText(
        "Statistics include test accounts. User list shows all accounts.",
      ),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByText("View daily data"));
    expect(
      screen.getByRole("table", { name: "Registrations by day" }),
    ).toHaveTextContent("0");
    expect(screen.getByText(/today is partial/)).toBeInTheDocument();
  });
  it("renders loading, retryable metric failure and zero registrations", async () => {
    vi.mocked(adminService.stats).mockRejectedValueOnce(new Error("offline"));
    setup();
    await screen.findByText("Could not load registration statistics.");
    const empty = {
      total_users: 0,
      registrations_7_days: 0,
      registrations_30_days: 0,
      daily_registrations: [{ date: "2026-10-29", count: 0 }],
      days: 30 as const,
      include_test: false,
      timezone: "Europe/Kyiv",
      as_of: "2026-10-29T10:00:00Z",
    };
    vi.mocked(adminService.stats).mockResolvedValue(empty);
    await userEvent.click(
      screen.getByRole("button", { name: "Retry statistics" }),
    );
    await screen.findByText("No registrations in this period.");
    expect(
      screen.queryByRole("status", { name: "Loading registration statistics" }),
    ).not.toBeInTheDocument();
  });
  it("marks and unmarks only through the dedicated action, with pending and error recovery", async () => {
    let release:
      ((value: { saved: boolean; is_test: boolean }) => void) | undefined;
    vi.mocked(adminService.setTest).mockReturnValueOnce(
      new Promise((resolve) => {
        release = resolve;
      }),
    );
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    const mark = await screen.findByRole("button", {
      name: "Mark as test account",
    });
    fireEvent.click(mark);
    fireEvent.click(mark);
    await waitFor(() => expect(adminService.setTest).toHaveBeenCalledTimes(1));
    expect(adminService.setTest).toHaveBeenCalledWith(user.id, true);
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
    vi.mocked(adminService.detail).mockResolvedValue({
      ...detail,
      user: { ...user, is_test: true },
    });
    release?.({ saved: true, is_test: true });
    const unmark = await screen.findByRole("button", {
      name: "Remove test account mark",
    });
    expect(unmark).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Test")).toBeInTheDocument();
    vi.mocked(adminService.setTest).mockRejectedValueOnce(new Error("offline"));
    await userEvent.click(unmark);
    await screen.findByRole("alert");
    expect(
      screen.getByRole("button", { name: "Remove test account mark" }),
    ).toBeEnabled();
    expect(adminService.update).not.toHaveBeenCalled();
    expect(adminService.grant).not.toHaveBeenCalled();
  });
});

describe("Admin access and user list", () => {
  it("denies access without fetching any user data", async () => {
    vi.mocked(adminService.access).mockRejectedValue(new Error("Forbidden"));
    setup();
    expect(
      await screen.findByRole("heading", {
        name: "Administrator access required",
      }),
    ).toBeInTheDocument();
    expect(adminService.users).not.toHaveBeenCalled();
  });
  it("renders access and row loading states, searches and pages", async () => {
    let release: ((value: { allowed: boolean }) => void) | undefined;
    vi.mocked(adminService.access).mockReturnValue(
      new Promise((resolve) => {
        release = resolve;
      }),
    );
    setup();
    expect(
      screen.getByRole("status", { name: "Checking administrator access" }),
    ).toBeInTheDocument();
    release?.({ allowed: true });
    const input = await screen.findByRole("textbox", {
      name: "Search users by name or email",
    });
    await screen.findByRole("button", { name: /Admin.*admin@example.com/ });
    await userEvent.type(input, " admin@example.com ");
    await userEvent.click(screen.getByRole("button", { name: "Search" }));
    await waitFor(() =>
      expect(adminService.users).toHaveBeenLastCalledWith(
        "admin@example.com",
        1,
        expect.any(AbortSignal),
      ),
    );
    await userEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() =>
      expect(adminService.users).toHaveBeenLastCalledWith(
        "admin@example.com",
        2,
        expect.any(AbortSignal),
      ),
    );
  });
  it("shows an empty result and a retryable error preserving typed search", async () => {
    vi.mocked(adminService.users)
      .mockResolvedValueOnce({ users: [], total: 0, page: 1, limit: 20 })
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ users: [], total: 0, page: 1, limit: 20 });
    setup();
    expect(
      await screen.findByText("No users found. Try another name or email."),
    ).toBeInTheDocument();
    await userEvent.type(screen.getByRole("textbox"), "missing");
    await userEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Could not load users",
    );
    expect(screen.getByRole("textbox")).toHaveValue("missing");
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(
      await screen.findByText("No users found. Try another name or email."),
    ).toBeInTheDocument();
  });
  it("localizes the admin access screen in Russian", async () => {
    await i18n.changeLanguage("ru");
    vi.mocked(adminService.access).mockRejectedValue(new Error("Forbidden"));
    setup();
    expect(
      await screen.findByRole("heading", {
        name: "Требуется доступ администратора",
      }),
    ).toBeInTheDocument();
  });
});
describe("Admin user edits and manual allowance", () => {
  it("saves only safe profile fields and blocks duplicate submits", async () => {
    let release: ((value: unknown) => void) | undefined;
    vi.mocked(adminService.update).mockReturnValue(
      new Promise((resolve) => {
        release = resolve;
      }),
    );
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    const name = await screen.findByLabelText("Name");
    await userEvent.clear(name);
    await userEvent.type(name, "Safe Name");
    await userEvent.selectOptions(screen.getByLabelText("Locale"), "uk");
    const submit = screen.getByRole("button", { name: "Save profile" });
    fireEvent.submit(submit.closest("form") as HTMLFormElement);
    fireEvent.submit(submit.closest("form") as HTMLFormElement);
    await waitFor(() => expect(adminService.update).toHaveBeenCalledTimes(1));
    expect(adminService.update).toHaveBeenCalledWith(user.id, {
      name: "Safe Name",
      locale: "uk",
    });
    release?.({ saved: true });
    expect(await screen.findByText("Profile saved.")).toBeInTheDocument();
  });
  it("retries an unknown grant outcome with the same immutable receipt", async () => {
    vi.mocked(adminService.grant)
      .mockRejectedValueOnce(new Error("timeout"))
      .mockResolvedValueOnce({ saved: true });
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    await userEvent.type(
      await screen.findByLabelText("Reason for granting"),
      "Support trial",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Could not save",
    );
    const first = vi.mocked(adminService.grant).mock.calls[0][1];
    expect(screen.getByLabelText("Days")).toBeDisabled();
    await userEvent.click(
      screen.getByRole("button", { name: "Retry same grant" }),
    );
    await waitFor(() => expect(adminService.grant).toHaveBeenCalledTimes(2));
    expect(vi.mocked(adminService.grant).mock.calls[1][1]).toEqual(first);
  });
  it("unlocks a rejected validation draft without clearing input and uses a new corrected receipt", async () => {
    vi.mocked(adminService.grant)
      .mockRejectedValueOnce(
        new ApiError("Invalid reason", "INVALID_REQUEST", 400),
      )
      .mockResolvedValueOnce({ saved: true });
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    const reason = await screen.findByLabelText("Reason for granting");
    fireEvent.change(reason, { target: { value: "Support\ttrial" } });
    await userEvent.click(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "The grant was rejected",
    );
    const rejected = vi.mocked(adminService.grant).mock.calls[0][1];
    expect(reason).toHaveValue("Support\ttrial");
    expect(reason).toBeEnabled();
    expect(screen.getByLabelText("Days")).toBeEnabled();
    fireEvent.change(reason, { target: { value: "Support trial" } });
    await userEvent.click(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    );
    await waitFor(() => expect(adminService.grant).toHaveBeenCalledTimes(2));
    const corrected = vi.mocked(adminService.grant).mock.calls[1][1];
    expect(corrected.reason).toBe("Support trial");
    expect(corrected.id).not.toBe(rejected.id);
  });
  it("reconciles a grant conflict into the existing active allowance without retrying the rejected grant", async () => {
    const activeGrant = {
      id: "00000000-0000-4000-8000-000000000003",
      period_start: "2026-10-29T00:00:00Z",
      period_end: "2026-11-28T00:00:00Z",
      revoked_at: null,
      actor_id: user.id,
      reason: "Another admin granted access",
      revoked_by: null,
      revoke_reason: null,
      credits: 500,
      used: 7,
      active: true,
    };
    vi.mocked(adminService.detail)
      .mockResolvedValueOnce(detail)
      .mockResolvedValueOnce({
        ...detail,
        effective_plan: "pro",
        manual_grants: [activeGrant],
      });
    vi.mocked(adminService.grant).mockRejectedValueOnce(
      new ApiError("Active grant exists", "CONFLICT", 409),
    );
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    await userEvent.type(
      await screen.findByLabelText("Reason for granting"),
      "Support trial",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    );
    expect(
      await screen.findByLabelText("Reason for revoking"),
    ).toBeInTheDocument();
    expect(await screen.findByText(/Details refreshed/)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Retry same grant" }),
    ).not.toBeInTheDocument();
    expect(adminService.detail).toHaveBeenCalledTimes(2);
    expect(adminService.grant).toHaveBeenCalledTimes(1);
    expect(
      screen.getByText("Another admin granted access"),
    ).toBeInTheDocument();
  });
  it("preserves the conflicting draft until an explicit access refresh succeeds", async () => {
    vi.mocked(adminService.detail)
      .mockResolvedValueOnce(detail)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(detail);
    vi.mocked(adminService.grant).mockRejectedValueOnce(
      new ApiError("Conflict", "CONFLICT", 409),
    );
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    await userEvent.type(
      await screen.findByLabelText("Reason for granting"),
      "Support trial",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "current details could not be loaded",
    );
    expect(screen.getByLabelText("Reason for granting")).toHaveValue(
      "Support trial",
    );
    expect(screen.getByLabelText("Reason for granting")).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Retry same grant" }),
    ).toBeDisabled();
    await userEvent.click(
      screen.getByRole("button", { name: "Refresh current access" }),
    );
    await waitFor(() =>
      expect(screen.getByLabelText("Reason for granting")).toBeEnabled(),
    );
    expect(screen.getByLabelText("Reason for granting")).toHaveValue(
      "Support trial",
    );
    expect(
      screen.getByRole("button", { name: "Grant manual Pro access" }),
    ).toBeEnabled();
    expect(adminService.grant).toHaveBeenCalledTimes(1);
  });
  it("revokes a specific manual grant while showing provider info read-only", async () => {
    const grant = {
      id: "00000000-0000-4000-8000-000000000003",
      period_start: "2026-10-29T00:00:00Z",
      period_end: "2026-11-28T00:00:00Z",
      revoked_at: null,
      actor_id: user.id,
      reason: "Trial",
      revoked_by: null,
      revoke_reason: null,
      credits: 500,
      used: 7,
      active: true,
    };
    vi.mocked(adminService.detail).mockResolvedValue({
      ...detail,
      provider_subscription_id: "creem-fixture-sub",
      manual_grants: [grant],
    });
    setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
    await userEvent.type(
      await screen.findByLabelText("Reason for revoking"),
      "Trial ended",
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Revoke manual access" }),
    );
    await waitFor(() =>
      expect(adminService.revoke).toHaveBeenCalledWith(
        user.id,
        grant.id,
        "Trial ended",
      ),
    );
    expect(screen.getByText(/creem-fixture-sub/)).toBeInTheDocument();
    expect(
      screen.queryByRole("textbox", { name: /email/i }),
    ).not.toBeInTheDocument();
  });
});

describe("Admin audit presentation regressions", () => {
  it("uses contrastful name text and restores the selected opener and scroll on Close", async () => {
    setup();
    const opener = await screen.findByRole("button", {
      name: /Admin.*admin@example.com/,
    });
    expect(opener).toHaveClass("text-foreground");
    const scroll = vi.fn();
    opener.scrollIntoView = scroll;
    await userEvent.click(opener);
    await screen.findByRole("heading", { name: "User details" });
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(opener).toHaveFocus();
    expect(scroll).toHaveBeenCalledWith({ block: "center" });
  });
  it("restores focus to the users region if pagination removed the opener", async () => {
    setup();
    const opener = await screen.findByRole("button", {
      name: /Admin.*admin@example.com/,
    });
    await userEvent.click(opener);
    await screen.findByRole("heading", { name: "User details" });
    vi.mocked(adminService.users).mockResolvedValue({
      users: [{ ...user, id: "other", name: "Other" }],
      total: 21,
      page: 2,
      limit: 20,
    });
    await userEvent.click(screen.getByRole("button", { name: "Next" }));
    await screen.findByRole("button", { name: /Other/ });
    expect(opener.isConnected).toBe(false);
    const region = screen.getByRole("region", { name: "Users" });
    const scroll = vi.fn();
    region.scrollIntoView = scroll;
    await userEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(region).toHaveFocus();
    expect(scroll).toHaveBeenCalledWith({ block: "center" });
  });
  it.each([
    ["en", "Scheduled end:"],
    ["uk", "Заплановане завершення:"],
    ["ru", "Плановое окончание:"],
  ])(
    "labels scheduled end separately from actual revocation in %s",
    async (locale, label) => {
      await i18n.changeLanguage(locale);
      const end = "2026-11-28T00:00:00Z";
      const revoked = "2026-10-29T10:11:12Z";
      vi.mocked(adminService.detail).mockResolvedValue({
        ...detail,
        manual_grants: [
          {
            id: "revoked",
            period_start: "2026-10-28T00:00:00Z",
            period_end: end,
            revoked_at: revoked,
            actor_id: user.id,
            reason: "Test grant",
            revoked_by: user.id,
            revoke_reason: "Test revoke",
            credits: 500,
            used: 0,
            active: false,
          },
        ],
      });
      setup(<AdminUserDetail id={user.id} onClose={vi.fn()} />);
      const header = await screen.findByText((text) => text.includes(label));
      expect(header).toHaveTextContent(new Date(end).toLocaleString());
      expect(header).not.toHaveTextContent(new Date(revoked).toLocaleString());
      expect(
        screen.getByText((text) => text.includes("Test revoke")),
      ).toHaveTextContent(new Date(revoked).toLocaleString());
    },
  );
});
