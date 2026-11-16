import { StrictMode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import { useAuthStore } from "@/stores/authStore";
import en from "@/shared/locales/en.json";

const billing = vi.hoisted(() => ({ getSubscription: vi.fn(), getCheckoutConfig: vi.fn(), createCheckoutSession: vi.fn() }));
vi.mock("@/services/subscriptionService", () => ({ subscriptionService: billing }));
vi.mock("@/shared/lib/features", () => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/shared/lib/analytics", () => ({ trackConversion: vi.fn(), trackBeginCheckout: vi.fn() }));
vi.mock("@/features/onboarding/useOnboarding", () => ({ useOnboarding: () => ({ shouldShow: false, complete: vi.fn() }) }));
vi.mock("@/features/onboarding/WelcomeWizard", () => ({ WelcomeWizard: () => null }));
vi.mock("@/widgets/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("@/widgets/Header", () => ({ Header: () => null }));
vi.mock("@/features/support/SupportButton", () => ({ SupportButton: () => null }));
vi.mock("@/features/subscription/components/SubscriptionSuccessModal", () => ({ SubscriptionSuccessModal: () => null }));
import OAuthAuthorizePage from "./OAuthAuthorize";
import { AppLayout } from "@/app/layouts/AppLayout";

const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en } }, interpolation: { escapeValue: false } });
const search = "?response_type=code&client_id=client&redirect_uri=https%3A%2F%2Fassistant.example%2Fcb&code_challenge=test&code_challenge_method=S256&state=kept&scope=mcp";
const consentPath = `/oauth/authorize${search}`;
const checkoutUrl = "https://www.creem.io/checkout/test";
const preview = { client: { client_id: "client", name: "Assistant", redirect_host: "assistant.example", is_metadata_document: false }, scope: "mcp", resource: "https://cursopus.com/api/mcp", plan_allowed: false };
let assign: ReturnType<typeof vi.fn>;
let fetchPreview: ReturnType<typeof vi.fn<(request: Request) => Promise<Response>>>;
let previewBody: unknown;
let previewStatus: number;

function Location() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname + location.search}</output>;
}
function mount(path: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const view = render(<StrictMode><I18nextProvider i18n={i18n}><QueryClientProvider client={client}>
    <MemoryRouter initialEntries={[path]}><Routes>
      <Route path="/oauth/authorize" element={<OAuthAuthorizePage />} />
      <Route path="/app" element={<AppLayout />}><Route path="settings" element={<p>Settings</p>} /></Route>
    </Routes><Location /></MemoryRouter>
  </QueryClientProvider></I18nextProvider></StrictMode>);
  return { ...view, client };
}
beforeEach(() => {
  vi.clearAllMocks();
  sessionStorage.clear();
  useAuthStore.setState({ isAuthenticated: true, user: { id: "user-1", email: "user@example.com", name: "User", locale: "en", created_at: "2026-01-01T00:00:00Z" } });
  previewBody = preview;
  previewStatus = 200;
  fetchPreview = vi.fn(async () => new Response(JSON.stringify(previewBody), {
    status: previewStatus, headers: { "content-type": "application/json" },
  }));
  vi.stubGlobal("fetch", fetchPreview);
  billing.getSubscription.mockResolvedValue({ plan: "free" });
  billing.getCheckoutConfig.mockResolvedValue({ plans: ["pro"] });
  billing.createCheckoutSession.mockResolvedValue({ checkout_url: checkoutUrl });
  assign = vi.fn();
  vi.stubGlobal("location", { ...window.location, pathname: "/oauth/authorize", search, hash: "", assign });
});
afterEach(() => { cleanup(); sessionStorage.clear(); vi.unstubAllGlobals(); });

async function startCheckout(path = consentPath) {
  const view = mount(path);
  await userEvent.click(await screen.findByRole("button", { name: en.settings.aiAssistants.upgrade }));
  const dialog = screen.getByRole("dialog", { name: en.settings.subscription.pricing.modalTitle });
  const choose = within(dialog).getByRole("button", { name: en.settings.subscription.pricing.choosePlan });
  await waitFor(() => expect(choose).toBeEnabled());
  await userEvent.click(choose);
  await waitFor(() => expect(assign).toHaveBeenCalledWith(checkoutUrl));
  view.unmount();
  return view;
}

describe("OAuth consent through hosted checkout", () => {
  it("resumes consent when Pro is already active on the checkout return", async () => {
    await startCheckout();
    billing.getSubscription.mockResolvedValue({ plan: "pro" });
    previewBody = { ...preview, plan_allowed: true };
    mount("/app/settings?subscription=success");
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
    expect(screen.getByTestId("location")).toHaveTextContent(consentPath);
  });

  it("returns from Settings to the original consent and rechecks eligibility after the backend confirms Pro", async () => {
    await startCheckout();
    const view = mount("/app/settings?subscription=success");
    expect(screen.getByTestId("location")).toHaveTextContent("/app/settings");
    expect(screen.queryByRole("button", { name: "Allow" })).not.toBeInTheDocument();
    await waitFor(() => expect(view.client.getQueryState(["subscription", "user-1"])?.fetchStatus).toBe("idle"));
    previewBody = { ...preview, plan_allowed: true };
    billing.getSubscription.mockResolvedValue({ plan: "pro" });
    await act(async () => { await view.client.invalidateQueries({ queryKey: ["subscription"] }); });
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
    expect(screen.getByTestId("location")).toHaveTextContent(consentPath);
    expect(fetchPreview.mock.calls.every(([request]) => request.url.endsWith(search) && request.method === "GET")).toBe(true);
    expect(sessionStorage.getItem("oauth_checkout_consent")).toBeNull();
    view.unmount();
    const later = mount("/app/settings?subscription=success");
    await waitFor(() => expect(later.client.getQueryState(["subscription", "user-1"])?.fetchStatus).toBe("idle"));
    expect(screen.getByTestId("location")).toHaveTextContent("/app/settings");
  });

  it("does not resume consent when checkout fails to start", async () => {
    billing.createCheckoutSession.mockRejectedValue(new Error("offline"));
    mount(consentPath);
    await userEvent.click(await screen.findByRole("button", { name: en.settings.aiAssistants.upgrade }));
    const dialog = screen.getByRole("dialog");
    const choose = within(dialog).getByRole("button", { name: en.settings.subscription.pricing.choosePlan });
    await waitFor(() => expect(choose).toBeEnabled());
    await userEvent.click(choose);
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(en.settings.subscription.checkoutError);
    expect(assign).not.toHaveBeenCalled();
    expect(sessionStorage.getItem("oauth_checkout_consent")).toBeNull();
  });

  it("re-runs a paid-feature preview when pricing closes after an upgrade", async () => {
    previewStatus = 403;
    previewBody = { error_code: "PAID_FEATURE" };
    mount(consentPath);
    await userEvent.click(await screen.findByRole("button", { name: en.settings.aiAssistants.upgrade }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    previewStatus = 200;
    previewBody = { ...preview, plan_allowed: true };
    await userEvent.keyboard("{Escape}");
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
    expect(assign).not.toHaveBeenCalled();
  });

  it.each(["uncached", "expired"])("shows upgrade without trusting the %s CIMD client and resumes after checkout", async (cacheState) => {
    const clientId = `https://assistant.example/${cacheState}.json`;
    const cimdSearch = search.replace("client_id=client", `client_id=${encodeURIComponent(clientId)}`);
    const path = `/oauth/authorize${cimdSearch}`;
    previewStatus = 403;
    previewBody = { error_code: "PAID_FEATURE" };
    vi.stubGlobal("location", { ...window.location, pathname: "/oauth/authorize", search: cimdSearch, hash: "", assign });
    const initial = mount(path);
    expect(await screen.findByText(en.settings.aiAssistants.upgradeDescription)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Allow" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Deny" })).not.toBeInTheDocument();
    expect(screen.queryByText(/Assistant wants to connect/)).not.toBeInTheDocument();
    expect(screen.queryByText("assistant.example")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to app" })).toHaveAttribute("href", "/app");
    expect(assign).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: en.settings.aiAssistants.upgrade }));
    const choose = within(screen.getByRole("dialog")).getByRole("button", { name: en.settings.subscription.pricing.choosePlan });
    await waitFor(() => expect(choose).toBeEnabled());
    await userEvent.click(choose);
    await waitFor(() => expect(assign).toHaveBeenCalledWith(checkoutUrl));
    expect(JSON.parse(sessionStorage.getItem("oauth_checkout_consent") ?? "null")).toMatchObject({ path });
    initial.unmount();
    billing.getSubscription.mockResolvedValue({ plan: "pro" });
    previewStatus = 200;
    previewBody = { ...preview, client: { ...preview.client, client_id: clientId, is_metadata_document: true }, plan_allowed: true };
    mount("/app/settings?subscription=success");
    expect(await screen.findByRole("button", { name: "Allow" })).toBeEnabled();
    expect(screen.getByTestId("location")).toHaveTextContent(path);
    expect(fetchPreview.mock.calls.every(([request]) => request.url.endsWith(cimdSearch) && request.method === "GET")).toBe(true);
    expect(fetchPreview.mock.calls.length).toBeGreaterThanOrEqual(2);
    expect(assign.mock.calls).toEqual([[checkoutUrl]]);
    expect(sessionStorage.getItem("oauth_checkout_consent")).toBeNull();
  });
});
