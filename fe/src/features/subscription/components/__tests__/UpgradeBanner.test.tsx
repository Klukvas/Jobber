import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { UpgradeBanner } from "../UpgradeBanner";

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
  limits: {
    max_jobs: 10,
    max_resumes: 5,
    max_applications: 20,
    max_ai_requests: 50,
    max_resume_builders: 3,
    max_cover_letters: 3,
  },
  nextPlan: "pro" as string | null,
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => subscription,
}));

const featuresMock = vi.hoisted(() => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/shared/lib/features", () => featuresMock);

// Stubbed rather than rendered: the real modal drags in the checkout hooks and
// a query client, and what is under test here is the wiring — that the CTA
// opens it, and that closing it takes it back off the page.
vi.mock("@/features/subscription/components/PricingModal", () => ({
  PricingModal: ({
    open,
    onOpenChange,
  }: {
    open: boolean;
    onOpenChange: (next: boolean) => void;
  }) => (
    <div data-testid="pricing-modal" data-open={String(open)}>
      <button type="button" onClick={() => onOpenChange(false)}>
        close pricing
      </button>
    </div>
  ),
}));

function upgradeButton() {
  return screen.getByRole("button", {
    name: /settings\.subscription\.upgradeTo/,
  });
}

describe("UpgradeBanner", () => {
  beforeEach(() => {
    featuresMock.FEATURES.PAYMENTS = true;
    subscription.limits = {
      max_jobs: 10,
      max_resumes: 5,
      max_applications: 20,
      max_ai_requests: 50,
      max_resume_builders: 3,
      max_cover_letters: 3,
    };
    subscription.nextPlan = "pro";
  });

  it("renders limit message for jobs", () => {
    render(<UpgradeBanner resource="jobs" />);
    expect(
      screen.getByText(/settings.subscription.limitReachedJobs/),
    ).toBeInTheDocument();
  });

  it("renders limit message for resumes", () => {
    render(<UpgradeBanner resource="resumes" />);
    expect(
      screen.getByText(/settings.subscription.limitReachedResumes/),
    ).toBeInTheDocument();
  });

  it("offers a working upgrade control, not just an instruction to upgrade", () => {
    render(<UpgradeBanner resource="jobs" />);
    expect(
      screen.getByRole("button", {
        name: "settings.subscription.upgradeToPro",
      }),
    ).toBeEnabled();
  });

  it("names the plan the customer would move to", () => {
    render(<UpgradeBanner resource="jobs" />);
    expect(
      screen.queryByRole("button", {
        name: "settings.subscription.upgradeToEnterprise",
      }),
    ).not.toBeInTheDocument();
  });

  // Only AI requests are counted per calendar month; every other allowance is
  // a lifetime cap, so quoting a reset date for those would be untrue.
  it("quotes a reset date for the monthly AI allowance", () => {
    render(<UpgradeBanner resource="ai" />);
    expect(
      screen.getByText(/settings.subscription.resetsOn/),
    ).toBeInTheDocument();
  });

  it.each(["jobs", "resumes", "resume_builders", "cover_letters"] as const)(
    "does not promise a monthly reset for %s",
    (resource) => {
      render(<UpgradeBanner resource={resource} />);
      expect(
        screen.queryByText(/settings.subscription.resetsOn/),
      ).not.toBeInTheDocument();
    },
  );

  /**
   * The AI banner used to read "AI match scoring is not available on your
   * current plan" — directly above "your monthly allowance resets on …". Both
   * halves were wrong: the free plan *has* an AI allowance, it comes back every
   * month, and it is spent by resume assistance and imports as much as by match
   * scoring. The message names the allowance and its size instead.
   */
  it("names the monthly AI allowance, with the number of requests in it", () => {
    render(<UpgradeBanner resource="ai" />);

    expect(
      screen.getByText(/settings\.subscription\.limitReachedAIMonthly/),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/settings\.subscription\.limitReachedAI$/),
    ).not.toBeInTheDocument();
  });

  it("does not tell an AI customer the feature is missing from their plan", () => {
    render(<UpgradeBanner resource="ai" />);

    // The old key is gone from the locales entirely; this guards the wiring.
    expect(
      screen.queryByText("settings.subscription.limitReachedAI"),
    ).not.toBeInTheDocument();
  });

  /**
   * Cover letters are the one resource that is absent on one plan and merely
   * finite on another, so the two states need different sentences: "not
   * included" at a cap of zero, "you have used all N" above it.
   */
  it("says cover letters are not included when the plan has none", () => {
    subscription.limits = { ...subscription.limits, max_cover_letters: 0 };
    render(<UpgradeBanner resource="cover_letters" />);

    expect(
      screen.getByText("settings.subscription.limitReachedCoverLetters"),
    ).toBeInTheDocument();
  });

  it("says the cover letters have been used up when the plan has some", () => {
    subscription.limits = { ...subscription.limits, max_cover_letters: 10 };
    render(<UpgradeBanner resource="cover_letters" />);

    expect(
      screen.getByText(/settings\.subscription\.limitReachedCoverLettersUsed/),
    ).toBeInTheDocument();
  });

  /**
   * The button is the banner's entire purpose: a customer who has hit a limit
   * has nowhere else to go from here. It rendered enabled and did nothing
   * observable in any test, so the modal could have been unwired — or mounted
   * open on every banner — without a single failure.
   */
  it("does not mount the pricing modal until the customer asks for it", () => {
    render(<UpgradeBanner resource="jobs" />);

    expect(screen.queryByTestId("pricing-modal")).not.toBeInTheDocument();
  });

  it("opens the pricing modal from the upgrade button", () => {
    render(<UpgradeBanner resource="jobs" />);

    fireEvent.click(upgradeButton());

    expect(screen.getByTestId("pricing-modal")).toHaveAttribute(
      "data-open",
      "true",
    );
  });

  it("takes the pricing modal back off the page when it is dismissed", () => {
    render(<UpgradeBanner resource="jobs" />);
    fireEvent.click(upgradeButton());

    fireEvent.click(screen.getByText("close pricing"));

    expect(screen.queryByTestId("pricing-modal")).not.toBeInTheDocument();
  });

  it("opens the same modal from the enterprise-tier banner", () => {
    subscription.nextPlan = "enterprise";
    render(<UpgradeBanner resource="ai" />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "settings.subscription.upgradeToEnterprise",
      }),
    );

    expect(screen.getByTestId("pricing-modal")).toBeInTheDocument();
  });

  it("renders nothing when payments are unavailable", () => {
    featuresMock.FEATURES.PAYMENTS = false;
    const { container } = render(<UpgradeBanner resource="jobs" />);

    expect(container).toBeEmptyDOMElement();
  });
});
