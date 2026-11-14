import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import en from "@/shared/locales/en.json";
import { ApiError } from "@/services/api";
import type { ApiToken } from "@/services/apiTokensService";

const service = vi.hoisted(() => ({ list: vi.fn(), create: vi.fn(), revoke: vi.fn(), activity: vi.fn() }));
const subscription = vi.hoisted(() => ({ isPro: true, isLoading: false }));
const account = vi.hoisted(() => ({ user: { id: "user-1" } }));
vi.mock("@/services/apiTokensService", () => ({ apiTokensService: service }));
vi.mock("@/shared/lib/features", () => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => subscription }));
vi.mock("@/stores/authStore", () => ({ useAuthStore: (select: (state: typeof account) => unknown) => select(account) }));
vi.mock("@/features/subscription/components/PricingModal", () => ({ PricingModal: ({ open }: { open: boolean }) => open ? <div role="dialog" aria-label="Pricing plans" /> : null }));

import { AiAssistantsSection } from "./AiAssistantsSection";

const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en } }, interpolation: { escapeValue: false } });
const token: ApiToken = { id: "token-1", name: "Laptop", token_prefix: "jbr_pat_ab12", scopes: ["mcp"], last_used_at: null, expires_at: null, revoked_at: null, created_at: "2026-01-01T12:00:00Z" };
const plaintext = "jbr_pat_test_only_plaintext";

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const result = render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}><AiAssistantsSection /></QueryClientProvider></I18nextProvider>);
  return { ...result, client };
}

beforeEach(() => {
  vi.clearAllMocks();
  subscription.isPro = true;
  subscription.isLoading = false;
  account.user = { id: "user-1" };
  service.list.mockResolvedValue([]);
  service.activity.mockResolvedValue([]);
  service.create.mockResolvedValue({ ...token, token: plaintext });
  service.revoke.mockResolvedValue(undefined);
});
afterEach(cleanup);

async function createToken(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText("Token name"), "  Laptop  ");
  await user.click(screen.getByRole("button", { name: "Create token" }));
}

describe("AI assistants settings", () => {
  it("shows the free plan explanation and opens pricing without fetching paid data", async () => {
    subscription.isPro = false;
    const user = userEvent.setup();
    mount();
    expect(screen.getByText("Pro")).toBeInTheDocument();
    expect(screen.queryByRole("form")).not.toBeInTheDocument();
    expect(service.list).not.toHaveBeenCalled();
    expect(service.activity).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Upgrade plan" }));
    expect(screen.getByRole("dialog", { name: "Pricing plans" })).toBeInTheDocument();
  });

  it("validates trimmed names and rejects names longer than 100 characters", async () => {
    const user = userEvent.setup();
    mount();
    const input = screen.getByLabelText("Token name");
    await user.type(input, "   ");
    await user.click(screen.getByRole("button", { name: "Create token" }));
    expect(screen.getByRole("alert")).toHaveTextContent("1–100");
    expect(input).toHaveFocus();
    fireEvent.change(input, { target: { value: "x".repeat(101) } });
    await user.click(screen.getByRole("button", { name: "Create token" }));
    expect(service.create).not.toHaveBeenCalled();
  });

  it("guards pending creation, shows/copies the token once, then clears it and the snippets", async () => {
    const user = userEvent.setup();
    const copy = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    let finish: ((value: ApiToken & { token: string }) => void) | undefined;
    service.create.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    const { client } = mount();
    expect(screen.getByLabelText("Token expiry")).toHaveValue("90");
    await createToken(user);
    expect(screen.getByRole("button", { name: "Creating…" })).toBeDisabled();
    fireEvent.submit(screen.getByRole("form", { name: "Create token" }));
    expect(service.create).toHaveBeenCalledTimes(1);
    expect(service.create).toHaveBeenCalledWith({ name: "Laptop", expires_in_days: 90 }, expect.objectContaining({ signal: expect.any(AbortSignal) }));
    await act(async () => { finish?.({ ...token, token: plaintext }); });
    const panel = screen.getByRole("region", { name: "Save your new token" });
    expect(panel).toHaveFocus();
    expect(within(panel).getByText(plaintext)).toBeInTheDocument();
    expect(screen.getByText(/will not be shown again/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copy token" }));
    expect(copy).toHaveBeenCalledWith(plaintext);
    expect(within(panel).getByRole("button", { name: "Copy token" })).toHaveTextContent("Copied");
    expect(JSON.stringify(client.getQueryCache().getAll().map((query) => query.state))).not.toContain(plaintext);
    expect(client.getMutationCache().getAll()).toHaveLength(0);
    expect(JSON.stringify(localStorage)).not.toContain(plaintext);
    await user.click(screen.getByRole("button", { name: "I saved it — dismiss" }));
    expect(document.body.textContent).not.toContain(plaintext);
    expect(document.body.textContent).toContain("<YOUR_TOKEN>");
    await waitFor(() => expect(screen.getByLabelText("Token name")).toHaveFocus());
  });

  it("uses origin URL and token/placeholder in both copyable setup snippets", async () => {
    const user = userEvent.setup();
    const copy = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    mount();
    await user.click(screen.getByRole("button", { name: "Copy Claude Code setup" }));
    expect(copy).toHaveBeenLastCalledWith(`claude mcp add --transport http cursopus ${window.location.origin}/api/mcp --header "Authorization: Bearer <YOUR_TOKEN>"`);
    await createToken(user);
    await screen.findByRole("region", { name: "Save your new token" });
    await user.click(screen.getByRole("button", { name: "Copy Cursor and other MCP clients setup" }));
    expect(JSON.parse(copy.mock.calls.at(-1)?.[0] ?? "")).toEqual({ mcpServers: { cursopus: { url: `${window.location.origin}/api/mcp`, headers: { Authorization: `Bearer ${plaintext}` } } } });
  });

  it("supports never-expiring tokens and forgets plaintext on unmount", async () => {
    const user = userEvent.setup();
    const view = mount();
    await user.selectOptions(screen.getByLabelText("Token expiry"), "never");
    await createToken(user);
    await screen.findByRole("region", { name: "Save your new token" });
    expect(service.create).toHaveBeenCalledWith(expect.objectContaining({ expires_in_days: null }), expect.anything());
    view.unmount();
    mount();
    expect(document.body.textContent).not.toContain(plaintext);
  });

  it("aborts in-flight creation on unmount", async () => {
    const user = userEvent.setup();
    service.create.mockImplementation(() => new Promise(() => {}));
    const view = mount();
    await createToken(user);
    const options = service.create.mock.calls[0][1] as { signal: AbortSignal };
    view.unmount();
    expect(options.signal.aborted).toBe(true);
  });

  it.each([["API_TOKEN_LIMIT_REACHED", "limit of 10"], ["VALIDATION_ERROR", "Check the token name"], ["INTERNAL_ERROR", "Could not create"]])("shows localized create error %s and preserves the name", async (code, message) => {
    const user = userEvent.setup();
    service.create.mockRejectedValue(new ApiError("backend detail", code, 403));
    mount();
    await createToken(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(message);
    expect(screen.getByLabelText("Token name")).toHaveValue("  Laptop  ");
    expect(screen.queryByText("backend detail")).not.toBeInTheDocument();
  });

  it.each(["create", "list", "activity"] as const)("falls back to the upgrade state on PAID_FEATURE from %s", async (operation) => {
    const user = userEvent.setup();
    service[operation].mockRejectedValue(new ApiError("", "PAID_FEATURE", 403));
    mount();
    if (operation === "create") await createToken(user);
    expect(await screen.findByRole("button", { name: "Upgrade plan" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Token name")).not.toBeInTheDocument();
  });

  it("renders empty states for tokens and activity", async () => {
    mount();
    expect(await screen.findByText(/No tokens yet/)).toBeInTheDocument();
    expect(await screen.findByText(/No assistant calls yet/)).toBeInTheDocument();
  });

  it("renders active, expired and muted revoked tokens with dates and never values", async () => {
    service.list.mockResolvedValue([token, { ...token, id: "expired", name: "Old", expires_at: "2020-01-01T00:00:00Z" }, { ...token, id: "revoked", name: "Revoked laptop", revoked_at: "2026-01-02T00:00:00Z" }]);
    mount();
    expect(await screen.findByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Expired")).toBeInTheDocument();
    const revokedRow = screen.getByText("Revoked laptop").closest("li");
    expect(revokedRow).toHaveClass("text-muted-foreground");
    expect(screen.getByText("Revoked")).toBeInTheDocument();
    expect(screen.getAllByText("jbr_pat_ab12…")).toHaveLength(3);
    expect(screen.getAllByText("Jan 1, 2026")).toHaveLength(3);
    expect(screen.getAllByText("Never").length).toBeGreaterThan(1);
    expect(screen.queryByRole("button", { name: "Revoke Revoked laptop" })).not.toBeInTheDocument();
  });

  it("confirms revocation, supports cancellation, and prevents repeat submission", async () => {
    const user = userEvent.setup();
    service.list.mockResolvedValue([token]);
    let finish: (() => void) | undefined;
    service.revoke.mockImplementation(() => new Promise<void>((resolve) => { finish = resolve; }));
    mount();
    await user.click(await screen.findByRole("button", { name: "Revoke Laptop" }));
    const dialog = screen.getByRole("dialog", { name: "Revoke token?" });
    expect(dialog).toHaveFocus();
    expect(service.revoke).not.toHaveBeenCalled();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Revoke Laptop" }));
    await user.click(screen.getByRole("button", { name: "Revoke" }));
    expect(screen.getByRole("button", { name: "Revoking…" })).toBeDisabled();
    expect(service.revoke).toHaveBeenCalledTimes(1);
    await act(async () => { finish?.(); });
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(service.list.mock.calls.length).toBeGreaterThan(1);
  });

  it("focuses the stable list heading after refetch removes the revoked token's button", async () => {
    const user = userEvent.setup();
    service.list.mockResolvedValueOnce([token]).mockResolvedValue([{ ...token, revoked_at: "2026-11-13T12:00:00Z" }]);
    mount();
    await user.click(await screen.findByRole("button", { name: "Revoke Laptop" }));
    await user.click(screen.getByRole("button", { name: "Revoke" }));
    expect(await screen.findByText("Revoked")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Revoke Laptop" })).not.toBeInTheDocument();
    await waitFor(() => expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Your tokens" })));
  });

  it("shows activity with human error labels, relative time and deleted-token fallback", async () => {
    const base = { id: "a1", tool: "list_jobs", ok: true, error_code: null, duration_ms: 12, created_at: new Date().toISOString(), api_token_id: token.id, token_name: token.name };
    service.activity.mockResolvedValue([base, { ...base, id: "a2", tool: "tailor_resume", ok: false, error_code: "CREDIT_UNAVAILABLE", token_name: null }, { ...base, id: "a3", ok: false, error_code: "FUTURE_ERROR" }]);
    mount();
    expect(await screen.findByText("Success")).toBeInTheDocument();
    expect(screen.getByText("Error: Not enough AI credits")).toBeInTheDocument();
    expect(screen.getByText("Error: Request could not be completed")).toBeInTheDocument();
    expect(screen.getByText("Deleted token")).toBeInTheDocument();
    expect(screen.getAllByText(/ago/)).toHaveLength(3);
  });

  it("shows loading states and retries each failed list independently", async () => {
    const user = userEvent.setup();
    let failTokens: ((reason: unknown) => void) | undefined;
    let failActivity: ((reason: unknown) => void) | undefined;
    service.list.mockImplementationOnce(() => new Promise((_, reject) => { failTokens = reject; }));
    service.activity.mockImplementationOnce(() => new Promise((_, reject) => { failActivity = reject; }));
    mount();
    expect(screen.getAllByRole("status", { name: "Loading..." })).toHaveLength(2);
    await act(async () => { failTokens?.(new Error()); failActivity?.(new Error()); });
    expect(await screen.findByText("Could not load tokens.")).toBeInTheDocument();
    expect(screen.getByText("Could not load recent activity.")).toBeInTheDocument();
    await user.click(within(screen.getByRole("region", { name: "Your tokens" })).getByRole("button", { name: "Try Again" }));
    await user.click(within(screen.getByRole("region", { name: "Recent activity" })).getByRole("button", { name: "Try Again" }));
    expect(await screen.findByText(/No tokens yet/)).toBeInTheDocument();
    expect(await screen.findByText(/No assistant calls yet/)).toBeInTheDocument();
  });

  it("keeps cached tokens visible and reports a failed background refresh", async () => {
    service.list.mockResolvedValue([token]);
    const { client } = mount();
    expect(await screen.findByText("Laptop")).toBeInTheDocument();
    service.list.mockRejectedValue(new Error("offline"));
    await act(async () => {
      await client.invalidateQueries({ queryKey: ["api-tokens", "user-1"] });
    });
    expect(await screen.findByText("Could not load tokens.")).toBeInTheDocument();
    expect(screen.getByText("Laptop")).toBeInTheDocument();
  });

  it("shows a revoke failure in the dialog without losing the selected token", async () => {
    const user = userEvent.setup();
    service.list.mockResolvedValue([token]);
    service.revoke.mockRejectedValue(new ApiError("", "API_TOKEN_NOT_FOUND", 404));
    mount();
    await user.click(await screen.findByRole("button", { name: "Revoke Laptop" }));
    await user.click(screen.getByRole("button", { name: "Revoke" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("This token no longer exists.");
    expect(screen.getByRole("dialog")).toHaveTextContent("Laptop");
    expect(screen.getByRole("button", { name: "Revoke" })).toBeEnabled();
  });
});
