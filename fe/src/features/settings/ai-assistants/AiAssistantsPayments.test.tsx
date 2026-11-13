import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import en from "@/shared/locales/en.json";
import { ApiError } from "@/services/api";

const features = vi.hoisted(() => ({ PAYMENTS: false }));
const subscription = vi.hoisted(() => ({ isPro: false, isLoading: false, plan: "free" }));
const tokens = vi.hoisted(() => ({ list: vi.fn(), activity: vi.fn() }));
const billing = vi.hoisted(() => ({
  getCheckoutConfig: vi.fn(), createCheckoutSession: vi.fn(), changePlan: vi.fn(),
}));
vi.mock("@/shared/lib/features", () => ({ FEATURES: features }));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => subscription }));
vi.mock("@/services/apiTokensService", () => ({ apiTokensService: tokens }));
vi.mock("@/services/subscriptionService", () => ({ subscriptionService: billing }));
vi.mock("@/stores/authStore", () => ({
  useAuthStore: (select: (state: { user: { id: string } }) => unknown) => select({ user: { id: "user-1" } }),
}));

// Keep the actual PricingModal, usePlanSelection and useCheckout payment gate.
import { AiAssistantsSection } from "./AiAssistantsSection";

const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en } }, interpolation: { escapeValue: false } });

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}><AiAssistantsSection /></QueryClientProvider></I18nextProvider>);
}

beforeEach(() => {
  vi.clearAllMocks();
  features.PAYMENTS = false;
  subscription.isPro = false;
  tokens.list.mockResolvedValue([]);
  tokens.activity.mockResolvedValue([]);
  billing.getCheckoutConfig.mockResolvedValue({ plans: ["pro"] });
});
afterEach(cleanup);

describe("AI assistants with the real pricing modal", () => {
  it("shows the existing payments-disabled explanation without an inert upgrade button", () => {
    mount();
    expect(screen.getByText(en.settings.subscription.paymentsDisabled)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upgrade plan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(billing.getCheckoutConfig).not.toHaveBeenCalled();
  });

  it("uses the same disabled-payments state after a PAID_FEATURE response", async () => {
    subscription.isPro = true;
    tokens.list.mockRejectedValue(new ApiError("", "PAID_FEATURE", 403));
    mount();
    expect(await screen.findByText(en.settings.subscription.paymentsDisabled)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upgrade plan" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Token name")).not.toBeInTheDocument();
  });

  it("opens the real pricing dialog when payments are enabled", async () => {
    features.PAYMENTS = true;
    const user = userEvent.setup();
    mount();
    await user.click(screen.getByRole("button", { name: "Upgrade plan" }));
    expect(screen.getByRole("dialog", { name: en.settings.subscription.pricing.modalTitle })).toBeInTheDocument();
    expect(screen.queryByText(en.settings.subscription.paymentsDisabled)).not.toBeInTheDocument();
    expect(billing.getCheckoutConfig).toHaveBeenCalled();
  });
});
