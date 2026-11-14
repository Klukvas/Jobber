import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import { ApiError } from "@/services/api";
import en from "@/shared/locales/en.json";
import { useAuthStore } from "@/stores/authStore";
const service = vi.hoisted(() => ({ preview: vi.fn(), authorize: vi.fn() }));
const auth = vi.hoisted(() => ({ login: vi.fn(), register: vi.fn(), verifyEmail: vi.fn() }));
vi.mock("@/services/oauthService", async (original) => ({ ...await original<typeof import("@/services/oauthService")>(), oauthService: service }));
vi.mock("@/services/authService", () => ({ authService: auth }));
vi.mock("@/shared/lib/features", () => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/features/subscription/components/PricingModal", () => ({ PricingModal: ({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) => open ? <div role="dialog" aria-label="Pricing plans"><button onClick={() => onOpenChange(false)}>Close pricing</button></div> : null }));
import OAuthAuthorizePage from "./OAuthAuthorize";
const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en } }, interpolation: { escapeValue: false } });
const account = { id: "user-1", name: "User", email: "user@example.com", locale: "en", created_at: "2026-01-01T00:00:00Z" };
const search = "?response_type=code&client_id=client&redirect_uri=https%3A%2F%2Fassistant.example%2Fcb&code_challenge=test&code_challenge_method=S256&state=kept&scope=mcp";
const preview = { client: { client_id: "client", name: "My assistant", redirect_host: "assistant.example", is_metadata_document: false }, scope: "mcp", resource: "https://cursopus.com/api/mcp", plan_allowed: true };
let assign: ReturnType<typeof vi.fn>;
function Location() { const location = useLocation(); return <output data-testid="location">{location.pathname + location.search}</output>; }
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}><MemoryRouter initialEntries={[`/oauth/authorize${search}`]}><OAuthAuthorizePage /><Location /></MemoryRouter></QueryClientProvider></I18nextProvider>);
}
beforeEach(() => {
  vi.clearAllMocks();
  useAuthStore.setState({ user: account, isAuthenticated: true });
  service.preview.mockResolvedValue(preview);
  service.authorize.mockResolvedValue({ redirect_to: "https://assistant.example/cb?code=test-only-code" });
  auth.login.mockResolvedValue({ user: account });
  auth.register.mockResolvedValue({}); auth.verifyEmail.mockResolvedValue({});
  assign = vi.fn();
  vi.stubGlobal("location", { ...window.location, assign });
});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });

describe("OAuth consent", () => {
  it("shows client, unverified notice, host, all permissions and noindex", async () => {
    mount();
    expect(await screen.findByText("My assistant wants to connect to Cursopus")).toBeInTheDocument();
    expect(screen.getByText(en.oauth.unverified)).toBeInTheDocument();
    expect(screen.getByText("assistant.example")).toBeInTheDocument();
    Object.values(en.oauth.permissions).forEach((text) => expect(screen.getByText(text)).toBeInTheDocument());
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute("content", "noindex, nofollow");
    expect(screen.getByRole("button", { name: "Allow" })).toBeEnabled();
  });
  it.each(["localhost:5000", "127.0.0.1:5000", "[::1]:5000"])("warns for loopback %s", async (host) => {
    service.preview.mockResolvedValue({ ...preview, client: { ...preview.client, redirect_host: host } });
    mount(); expect(await screen.findByRole("note")).toHaveTextContent(en.oauth.loopbackWarning);
  });
  it("shows loading and a retryable error without following the requested redirect", async () => {
    let fail: ((error: Error) => void) | undefined;
    service.preview.mockImplementationOnce(() => new Promise((_, reject) => { fail = reject; }));
    mount();
    expect(screen.getByRole("status", { name: "Loading..." })).toBeInTheDocument();
    await act(async () => { fail?.(new ApiError("unsafe details", "INVALID_CLIENT", 400)); });
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.previewError);
    expect(assign).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Allow" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Try Again" }));
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
  });
  it("follows only the preview's trusted error redirect", async () => {
    service.preview.mockResolvedValue({ redirect_to: "https://assistant.example/cb?error=invalid_request" });
    mount(); await waitFor(() => expect(assign).toHaveBeenCalledWith("https://assistant.example/cb?error=invalid_request"));
    expect(service.authorize).not.toHaveBeenCalled();
  });
  it("disables Allow on Free, opens existing upgrade modal and still permits Deny", async () => {
    service.preview.mockResolvedValue({ ...preview, plan_allowed: false });
    mount();
    expect(await screen.findByRole("button", { name: "Allow" })).toBeDisabled();
    expect(screen.getByText(en.settings.aiAssistants.upgradeDescription)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Upgrade plan" }));
    expect(screen.getByRole("dialog", { name: "Pricing plans" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Deny" }));
    await waitFor(() => expect(service.authorize).toHaveBeenCalledWith(expect.objectContaining({ state: "kept" }), "deny", expect.objectContaining({ signal: expect.any(AbortSignal) })));
  });
  it("refreshes plan eligibility when the upgrade dialog closes", async () => {
    service.preview.mockResolvedValueOnce({ ...preview, plan_allowed: false });
    mount();
    await userEvent.click(await screen.findByRole("button", { name: "Upgrade plan" }));
    await userEvent.click(screen.getByRole("button", { name: "Close pricing" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Allow" })).toBeEnabled());
    expect(service.preview).toHaveBeenCalledTimes(2);
  });
  it.each(["Allow", "Deny"])("submits %s once, keeps buttons disabled through navigation and assigns the response", async (label) => {
    let finish: ((result: { redirect_to: string }) => void) | undefined;
    service.authorize.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    mount();
    const button = await screen.findByRole("button", { name: label });
    fireEvent.click(button); fireEvent.click(button);
    expect(service.authorize).toHaveBeenCalledTimes(1);
    expect(service.authorize).toHaveBeenCalledWith(expect.objectContaining({ state: "kept", code_challenge: "test" }), label.toLowerCase(), expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(screen.getByRole("button", { name: "Allow" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Deny" })).toBeDisabled();
    await act(async () => { finish?.({ redirect_to: "https://assistant.example/cb?result=test" }); });
    expect(assign).toHaveBeenCalledWith("https://assistant.example/cb?result=test");
    expect(button).toBeDisabled();
  });
  it("shows safe submission errors and allows retry", async () => {
    service.authorize.mockRejectedValueOnce(new Error("secret details"));
    mount();
    await userEvent.click(await screen.findByRole("button", { name: "Allow" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.submitError);
    expect(screen.getByRole("button", { name: "Allow" })).toBeEnabled();
    await userEvent.click(screen.getByRole("button", { name: "Allow" }));
    await waitFor(() => expect(assign).toHaveBeenCalled());
  });
  it.each(["preview", "authorize"] as const)("explains a trusted-origin rejection during %s without exposing backend details", async (stage) => {
    service[stage].mockRejectedValueOnce(new ApiError("private backend details", "ORIGIN_NOT_ALLOWED", 403));
    mount();
    if (stage === "authorize") await userEvent.click(await screen.findByRole("button", { name: "Allow" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.originError);
    expect(screen.queryByText("private backend details")).not.toBeInTheDocument();
    expect(assign).not.toHaveBeenCalled();
  });
  it("requests authentication again after a 401 and reloads preview after login", async () => {
    service.preview.mockRejectedValueOnce(new ApiError("", "UNAUTHORIZED", 401));
    mount();
    const dialog = await screen.findByRole("dialog", { name: "Login" });
    await userEvent.type(screen.getByLabelText("Email"), account.email);
    await userEvent.type(screen.getByLabelText("Password"), "password123");
    await userEvent.click(within(dialog).getByRole("button", { name: "Login" }));
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
    expect(service.preview).toHaveBeenCalledTimes(2);
  });
  it("does not render consent when embedded in another frame", () => {
    vi.stubGlobal("top", {});
    mount();
    expect(screen.getByRole("alert")).toHaveTextContent(en.oauth.frameError);
    expect(service.preview).not.toHaveBeenCalled();
  });
  it("aborts pending authorization and does not navigate after the user signs out", async () => {
    let finish: ((result: { redirect_to: string }) => void) | undefined;
    service.authorize.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    mount();
    await userEvent.click(await screen.findByRole("button", { name: "Allow" }));
    const options = service.authorize.mock.calls[0][2] as { signal: AbortSignal };
    act(() => useAuthStore.setState({ user: null, isAuthenticated: false }));
    expect(options.signal.aborted).toBe(true);
    await act(async () => { finish?.({ redirect_to: "https://assistant.example/cb?code=test" }); });
    expect(assign).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog", { name: "Login" })).toBeInTheDocument();
  });
  it("reuses login and retains the exact consent URL on authentication", async () => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
    mount();
    expect(screen.getByRole("dialog", { name: "Login" })).toBeInTheDocument();
    expect(service.preview).not.toHaveBeenCalled();
    await userEvent.type(screen.getByLabelText("Email"), account.email);
    await userEvent.type(screen.getByLabelText("Password"), "password123");
    await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Login" }));
    expect(await screen.findByRole("button", { name: "Allow" })).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent(`/oauth/authorize${search}`);
  });
  it("reuses registration and returns after verification/login without losing the consent URL", async () => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
    mount();
    await userEvent.click(screen.getByRole("button", { name: "Register" }));
    await userEvent.type(screen.getByLabelText("Email"), account.email);
    await userEvent.type(screen.getByLabelText("Password"), "password123");
    await userEvent.type(screen.getByLabelText("Confirm Password"), "password123");
    await userEvent.click(screen.getByRole("button", { name: "Register" }));
    await userEvent.type(await screen.findByLabelText("Verification code"), "123456");
    await userEvent.click(screen.getByRole("button", { name: "Verify" }));
    expect(await screen.findByRole("button", { name: "Allow" })).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent(`/oauth/authorize${search}`);
  });
});
