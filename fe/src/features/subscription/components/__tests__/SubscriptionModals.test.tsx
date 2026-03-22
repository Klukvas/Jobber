import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SubscriptionSuccessModal } from "../SubscriptionSuccessModal";
import { UpgradeModal } from "../UpgradeModal";
import { PricingModal } from "../PricingModal";

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

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: "free", nextPlan: "pro" }),
}));

vi.mock("@/features/subscription/usePaddleCheckout", () => ({
  usePaddleCheckout: () => ({ openCheckout: vi.fn(), isReady: true }),
}));

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

// ---------- SubscriptionSuccessModal ----------
describe("SubscriptionSuccessModal", () => {
  it("renders when plan is provided", () => {
    render(<SubscriptionSuccessModal plan="pro" onClose={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.upgradeSuccess.title"),
    ).toBeInTheDocument();
  });

  it("returns null when plan is null", () => {
    const { container } = render(
      <SubscriptionSuccessModal plan={null} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("calls onClose when CTA is clicked", () => {
    const onClose = vi.fn();
    render(<SubscriptionSuccessModal plan="pro" onClose={onClose} />);
    fireEvent.click(
      screen.getByText("settings.subscription.upgradeSuccess.cta"),
    );
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("renders description with plan label", () => {
    render(<SubscriptionSuccessModal plan="enterprise" onClose={vi.fn()} />);
    expect(
      screen.getByText(/settings.subscription.upgradeSuccess.description/),
    ).toBeInTheDocument();
  });
});

// ---------- UpgradeModal ----------
describe("UpgradeModal", () => {
  it("renders when open", () => {
    render(<UpgradeModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.aiLimitTitle"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <UpgradeModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders close and upgrade buttons", () => {
    render(<UpgradeModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("common.close")).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.upgradeForMore"),
    ).toBeInTheDocument();
  });
});

// ---------- PricingModal ----------
describe("PricingModal", () => {
  it("renders when open", () => {
    render(<PricingModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.pricing.modalTitle"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <PricingModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders all three plan cards", () => {
    render(<PricingModal open={true} onOpenChange={vi.fn()} />);
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
    render(<PricingModal open={true} onOpenChange={vi.fn()} />);
    // Free is the current plan, so it gets the current badge
    expect(
      screen.getByText("settings.subscription.pricing.currentPlanBadge"),
    ).toBeInTheDocument();
  });

  it("shows popular badge for pro plan", () => {
    render(<PricingModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.pricing.popular"),
    ).toBeInTheDocument();
  });
});
