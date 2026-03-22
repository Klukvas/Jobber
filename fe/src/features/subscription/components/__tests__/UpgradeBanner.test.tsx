import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    limits: {
      max_jobs: 10,
      max_resumes: 5,
      max_applications: 20,
      max_ai_requests: 50,
      max_resume_builders: 3,
      max_cover_letters: 3,
    },
    nextPlan: "pro",
  }),
}));

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

describe("UpgradeBanner", () => {
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

  it("renders payments disabled message", () => {
    render(<UpgradeBanner resource="jobs" />);
    expect(
      screen.getByText("settings.subscription.paymentsDisabled"),
    ).toBeInTheDocument();
  });
});
