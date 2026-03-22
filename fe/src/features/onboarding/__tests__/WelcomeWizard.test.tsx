import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { WelcomeWizard } from "../WelcomeWizard";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("@/stores/sidebarStore", () => ({
  useSidebarStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isExpanded: true }),
}));

vi.mock("../StepIndicator", () => ({
  StepIndicator: () => <div data-testid="step-indicator" />,
}));

vi.mock("../WizardStepContent", () => ({
  WizardStepContent: ({ step }: { step: number }) => (
    <div data-testid="step-content">Step {step}</div>
  ),
  TOTAL_STEPS: 8,
}));

vi.mock("../useOnboarding", () => ({
  setOnboardingHighlight: vi.fn(),
}));

describe("WelcomeWizard", () => {
  it("renders when open=true", () => {
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByTestId("step-content")).toBeInTheDocument();
  });

  it("returns null when open=false", () => {
    const { container } = render(
      <WelcomeWizard open={false} onComplete={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders navigation buttons", () => {
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);
    // On first step, should have "next" button
    expect(screen.getByText("onboarding.next")).toBeInTheDocument();
  });

  it("renders step indicator", () => {
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);
    expect(screen.getByTestId("step-indicator")).toBeInTheDocument();
  });
});
