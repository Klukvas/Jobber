import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { WizardStepContent, TOTAL_STEPS } from "../WizardStepContent";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

describe("WizardStepContent", () => {
  it("renders the welcome step (step 0)", () => {
    render(<WizardStepContent step={0} />);
    expect(screen.getByText("onboarding.welcome.title")).toBeInTheDocument();
    expect(
      screen.getByText("onboarding.welcome.subtitle"),
    ).toBeInTheDocument();
  });

  it("renders the company step (step 1)", () => {
    render(<WizardStepContent step={1} />);
    expect(
      screen.getByText("onboarding.steps.company.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("onboarding.steps.company.description"),
    ).toBeInTheDocument();
  });

  it("renders the resume step (step 2)", () => {
    render(<WizardStepContent step={2} />);
    expect(
      screen.getByText("onboarding.steps.resume.title"),
    ).toBeInTheDocument();
  });

  it("renders the job step (step 3)", () => {
    render(<WizardStepContent step={3} />);
    expect(
      screen.getByText("onboarding.steps.job.title"),
    ).toBeInTheDocument();
  });

  it("renders the done step (last step)", () => {
    render(<WizardStepContent step={7} />);
    expect(
      screen.getByText("onboarding.steps.done.title"),
    ).toBeInTheDocument();
  });

  it("exports TOTAL_STEPS as 8", () => {
    expect(TOTAL_STEPS).toBe(8);
  });
});
