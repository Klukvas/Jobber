import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { SubscriptionSuccessModal } from "../SubscriptionSuccessModal";
import { UpgradeModal } from "../UpgradeModal";
import { PricingModal } from "../PricingModal";
import type { SubscriptionPlan } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params) {
        return Object.entries(params).reduce(
          (acc, [k, v]) => acc.replace(`{{${k}}}`, String(v)),
          key,
        );
      }
      return key;
    },
    i18n: { language: "en" },
  }),
}));

const subscription = vi.hoisted(() => ({
  plan: "free" as SubscriptionPlan,
  nextPlan: "pro" as SubscriptionPlan | null,
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => subscription,
}));

const checkout = vi.hoisted(() => ({
  openCheckout: vi.fn(),
  isReady: true,
  isPending: false,
  error: null as Error | null,
}));

vi.mock("@/features/subscription/useCheckout", () => ({
  useCheckout: () => checkout,
}));

const api = vi.hoisted(() => ({
  changePlan: vi.fn(),
  createCheckoutSession: vi.fn(),
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: {
    changePlan: api.changePlan,
    createCheckoutSession: api.createCheckoutSession,
  },
}));

const notifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/lib/notifications", () => notifications);

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

// The modals drive a real useMutation, so they need a real client. Retries are
// off so a rejected mutation surfaces its error on the first attempt.
function renderWithClient(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  subscription.plan = "free";
  subscription.nextPlan = "pro";
  checkout.openCheckout = vi.fn();
  checkout.isReady = true;
  checkout.isPending = false;
  checkout.error = null;
  api.changePlan.mockResolvedValue(undefined);
});

// ---------- SubscriptionSuccessModal ----------
describe("SubscriptionSuccessModal", () => {
  it("renders when plan is provided", () => {
    renderWithClient(<SubscriptionSuccessModal plan="pro" onClose={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.upgradeSuccess.title"),
    ).toBeInTheDocument();
  });

  it("returns null when plan is null", () => {
    const { container } = renderWithClient(
      <SubscriptionSuccessModal plan={null} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("calls onClose when CTA is clicked", () => {
    const onClose = vi.fn();
    renderWithClient(<SubscriptionSuccessModal plan="pro" onClose={onClose} />);
    fireEvent.click(
      screen.getByText("settings.subscription.upgradeSuccess.cta"),
    );
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders description with plan label", () => {
    renderWithClient(
      <SubscriptionSuccessModal plan="enterprise" onClose={vi.fn()} />,
    );
    expect(
      screen.getByText(/settings.subscription.upgradeSuccess.description/),
    ).toBeInTheDocument();
  });
});

// ---------- UpgradeModal ----------
describe("UpgradeModal", () => {
  it("renders when open", () => {
    renderWithClient(<UpgradeModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.aiLimitTitle"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = renderWithClient(
      <UpgradeModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders close and upgrade buttons", () => {
    renderWithClient(<UpgradeModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("common.close")).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.upgradeForMore"),
    ).toBeInTheDocument();
  });
});

// ---------- PricingModal ----------
describe("PricingModal", () => {
  it("renders when open", () => {
    renderWithClient(<PricingModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.pricing.modalTitle"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = renderWithClient(
      <PricingModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders all three plan cards", () => {
    renderWithClient(<PricingModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.freePlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.proPlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.enterprisePlan"),
    ).toBeInTheDocument();
  });

  it("marks current plan with badge", () => {
    renderWithClient(<PricingModal open={true} onOpenChange={vi.fn()} />);
    // Free is the current plan, so it gets the current badge
    expect(
      screen.getByText("settings.subscription.pricing.currentPlanBadge"),
    ).toBeInTheDocument();
  });

  it("shows popular badge for pro plan", () => {
    renderWithClient(<PricingModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.pricing.popular"),
    ).toBeInTheDocument();
  });
});

// ---------- checkout failure surfacing ----------
describe("checkout failures stay visible", () => {
  beforeEach(() => {
    checkout.openCheckout = vi.fn();
    checkout.isReady = true;
    checkout.isPending = false;
    checkout.error = null;
  });

  it("PricingModal keeps itself open when starting checkout fails", () => {
    const onOpenChange = vi.fn();
    checkout.error = new Error("checkout unavailable");

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);

    expect(screen.getByRole("alert")).toHaveTextContent(
      "settings.subscription.checkoutError",
    );
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("PricingModal does not close the dialog on select", () => {
    // A successful checkout navigates away; closing here would hide a failure.
    const onOpenChange = vi.fn();
    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);

    fireEvent.click(
      screen.getAllByText("settings.subscription.pricing.choosePlan")[0],
    );

    expect(checkout.openCheckout).toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("UpgradeModal disables the CTA while a session is being created", () => {
    checkout.isPending = true;

    renderWithClient(<UpgradeModal open onOpenChange={vi.fn()} />);

    expect(
      screen
        .getByText("settings.subscription.upgradeForMore")
        .closest("button"),
    ).toBeDisabled();
  });
});

// ---------- the payment popup owns the keyboard while it is up ----------
describe("the modal yields the keyboard to the payment popup", () => {
  // The provider's popup is appended outside the modal's DOM. Escape reaching
  // the modal would close the page behind a payment already in flight.

  it("PricingModal ignores Escape while the checkout popup is open", () => {
    checkout.isPending = true;
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("UpgradeModal ignores Escape while the checkout popup is open", () => {
    checkout.isPending = true;
    const onOpenChange = vi.fn();

    renderWithClient(<UpgradeModal open onOpenChange={onOpenChange} />);
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("PricingModal closes on Escape as usual when no popup is open", () => {
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("a subscriber's in-flight plan change still closes on Escape", () => {
    // Nothing is drawn over the page for a plan change, so the modal keeps the
    // keyboard even while the request is pending.
    subscription.plan = "pro";
    checkout.isPending = true;
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

// ---------- a subscriber never checks out twice ----------
describe("an existing subscriber changes plan instead of buying again", () => {
  // The subscription row holds a single provider subscription ID. A second
  // checkout would overwrite it and leave the first subscription billing at the
  // provider with nothing pointing at it. The backend answers 409; the UI must
  // never get that far.

  it("PricingModal sends pro -> enterprise through change-plan, never checkout", async () => {
    subscription.plan = "pro";
    subscription.nextPlan = "enterprise";
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);

    fireEvent.click(
      screen.getByText("settings.subscription.pricing.switchPlan"),
    );

    await waitFor(() =>
      expect(api.changePlan).toHaveBeenCalledExactlyOnceWith("enterprise"),
    );
    expect(checkout.openCheckout).not.toHaveBeenCalled();
    expect(api.createCheckoutSession).not.toHaveBeenCalled();
  });

  it("UpgradeModal sends pro -> enterprise through change-plan, never checkout", async () => {
    subscription.plan = "pro";
    subscription.nextPlan = "enterprise";

    renderWithClient(<UpgradeModal open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByText("settings.subscription.upgradeForMore"));

    await waitFor(() =>
      expect(api.changePlan).toHaveBeenCalledExactlyOnceWith("enterprise"),
    );
    expect(checkout.openCheckout).not.toHaveBeenCalled();
    expect(api.createCheckoutSession).not.toHaveBeenCalled();
  });

  it("closes the modal and confirms once the plan change is accepted", async () => {
    subscription.plan = "pro";
    subscription.nextPlan = "enterprise";
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);

    fireEvent.click(
      screen.getByText("settings.subscription.pricing.switchPlan"),
    );

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(notifications.showSuccessNotification).toHaveBeenCalledWith(
      "settings.subscription.manage.planChanged",
    );
  });

  it("keeps the modal open and shows the failure when a plan change is refused", async () => {
    subscription.plan = "pro";
    subscription.nextPlan = "enterprise";
    api.changePlan.mockRejectedValue(new Error("provider refused"));
    const onOpenChange = vi.fn();

    renderWithClient(<PricingModal open onOpenChange={onOpenChange} />);

    fireEvent.click(
      screen.getByText("settings.subscription.pricing.switchPlan"),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "settings.subscription.manage.changePlanError",
    );
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("offers no free-plan CTA to a subscriber — that is a cancellation", async () => {
    // A button that can only fail is worse than no button: returning to free
    // means cancelling, which lives in the manage-subscription flow.
    subscription.plan = "pro";

    renderWithClient(<PricingModal open onOpenChange={vi.fn()} />);

    expect(
      screen.getByText("settings.subscription.pricing.downgradeViaCancel"),
    ).toBeInTheDocument();

    const ctas = screen.getAllByRole("button");
    for (const cta of ctas) {
      expect(cta).not.toHaveTextContent(
        "settings.subscription.pricing.downgradeViaCancel",
      );
    }

    // Only the enterprise card offers a switch; free is a note, pro is current.
    expect(
      screen.getAllByText("settings.subscription.pricing.switchPlan"),
    ).toHaveLength(1);
  });

  it("a free user still buys through checkout", async () => {
    subscription.plan = "free";

    renderWithClient(<PricingModal open onOpenChange={vi.fn()} />);

    fireEvent.click(
      screen.getAllByText("settings.subscription.pricing.choosePlan")[0],
    );

    expect(checkout.openCheckout).toHaveBeenCalledExactlyOnceWith("pro");
    expect(api.changePlan).not.toHaveBeenCalled();
  });
});

/**
 * A dialog with no accessible name is announced as just "dialog". Every one of
 * these draws a heading; the shared `Dialog`/`DialogTitle` pair is what turns
 * that heading into the name, and this is the consumer-side half of that
 * contract.
 */
function expectDialogNamedBy(headingText: string) {
  const dialog = screen.getByRole("dialog");
  const labelledBy = dialog.getAttribute("aria-labelledby");

  expect(labelledBy).toBeTruthy();
  expect(document.getElementById(labelledBy ?? "")?.textContent).toBe(
    headingText,
  );
}

describe("PricingModal — accessible name", () => {
  it("names the plan chooser by its own heading", () => {
    renderWithClient(<PricingModal open onOpenChange={vi.fn()} />);

    expectDialogNamedBy("settings.subscription.pricing.modalTitle");
  });
});
