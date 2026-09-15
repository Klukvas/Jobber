import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

// The real component renders the step's heading and takes the id the wizard
// uses to name itself; the stub has to do both, or the accessible-name tests
// below would be testing the stub rather than the wizard.
vi.mock("../WizardStepContent", () => ({
  WizardStepContent: ({
    step,
    headingId,
  }: {
    step: number;
    headingId?: string;
  }) => (
    <div data-testid="step-content">
      <h3 id={headingId}>Step {step}</h3>
    </div>
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

describe("WelcomeWizard — leaving the tour", () => {
  it("asks how to leave instead of completing on Escape", async () => {
    const onComplete = vi.fn();
    const onDismissForSession = vi.fn();
    render(
      <WelcomeWizard
        open
        onComplete={onComplete}
        onDismissForSession={onDismissForSession}
      />,
    );

    fireEvent.keyDown(document, { key: "Escape" });

    expect(screen.getByText("onboarding.exit.title")).toBeInTheDocument();
    expect(onComplete).not.toHaveBeenCalled();
    expect(onDismissForSession).not.toHaveBeenCalled();
  });

  it("asks how to leave instead of completing on a backdrop click", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    const { container } = render(
      <WelcomeWizard
        open
        onComplete={onComplete}
        onDismissForSession={vi.fn()}
      />,
    );

    const backdrop = container.querySelector(".fixed.inset-0.z-40");
    await user.click(backdrop as Element);

    expect(screen.getByText("onboarding.exit.title")).toBeInTheDocument();
    expect(onComplete).not.toHaveBeenCalled();
  });

  it("returns to the tour when the customer keeps going", async () => {
    const user = userEvent.setup();
    render(
      <WelcomeWizard open onComplete={vi.fn()} onDismissForSession={vi.fn()} />,
    );

    fireEvent.keyDown(document, { key: "Escape" });
    await user.click(screen.getByText("onboarding.exit.keepGoing"));

    expect(screen.getByTestId("step-content")).toBeInTheDocument();
    expect(screen.queryByText("onboarding.exit.title")).not.toBeInTheDocument();
  });

  it("a second Escape backs out of the question", () => {
    render(
      <WelcomeWizard open onComplete={vi.fn()} onDismissForSession={vi.fn()} />,
    );

    fireEvent.keyDown(document, { key: "Escape" });
    fireEvent.keyDown(document, { key: "Escape" });

    expect(screen.getByTestId("step-content")).toBeInTheDocument();
  });

  it("'not now' hides the tour without retiring it", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    const onDismissForSession = vi.fn();
    render(
      <WelcomeWizard
        open
        onComplete={onComplete}
        onDismissForSession={onDismissForSession}
      />,
    );

    fireEvent.keyDown(document, { key: "Escape" });
    await user.click(screen.getByText("onboarding.exit.notNow"));

    expect(onDismissForSession).toHaveBeenCalledTimes(1);
    expect(onComplete).not.toHaveBeenCalled();
  });

  it("'never again' is the only path that marks onboarding complete", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    const onDismissForSession = vi.fn();
    render(
      <WelcomeWizard
        open
        onComplete={onComplete}
        onDismissForSession={onDismissForSession}
      />,
    );

    fireEvent.keyDown(document, { key: "Escape" });
    await user.click(screen.getByText("onboarding.exit.neverAgain"));

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(onDismissForSession).not.toHaveBeenCalled();
  });

  it("the skip button also asks rather than deciding", async () => {
    const user = userEvent.setup();
    const onComplete = vi.fn();
    render(
      <WelcomeWizard
        open
        onComplete={onComplete}
        onDismissForSession={vi.fn()}
      />,
    );

    await user.click(screen.getByText("onboarding.skip"));

    expect(screen.getByText("onboarding.exit.title")).toBeInTheDocument();
    expect(onComplete).not.toHaveBeenCalled();
  });

  // The tour's two primary controls measured 36px tall on a phone — the wizard
  // covers the whole screen there, so they are the only things to press.
  it.each(["onboarding.skip", "onboarding.next"])(
    "gives the %s button a 44px minimum on phones",
    (label) => {
      render(<WelcomeWizard open onComplete={vi.fn()} />);

      expect(screen.getByText(label).className).toContain("max-sm:h-11");
    },
  );

  /**
   * The tour is a hand-rolled modal rather than a `Dialog`, and had no trap at
   * all: Tab walked straight into the app it is describing, behind a backdrop
   * that hides where focus went.
   */
  describe("keyboard containment", () => {
    it("starts with focus on the tour card, not on body", () => {
      render(<WelcomeWizard open onComplete={vi.fn()} />);

      expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(
        true,
      );
    });

    it("pulls Tab back in when focus starts outside", () => {
      render(
        <>
          <button>behind the tour</button>
          <WelcomeWizard open onComplete={vi.fn()} />
        </>,
      );
      screen.getByText("behind the tour").focus();

      fireEvent.keyDown(document, { key: "Tab" });

      expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(
        true,
      );
    });
  });
});

/**
 * The tour is one dialog showing eight different things, and it announced
 * "Welcome to Jobber" for every one of them — including the question about
 * leaving. The name now follows whichever heading is drawn.
 */
describe("WelcomeWizard — accessible name", () => {
  /** The dialog's name, resolved the way assistive tech resolves it. */
  function dialogName(): string | null {
    const labelledBy = screen
      .getByRole("dialog")
      .getAttribute("aria-labelledby");
    if (!labelledBy) return null;
    return document.getElementById(labelledBy)?.textContent ?? null;
  }

  it("names the dialog after the first step's heading", () => {
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);

    expect(dialogName()).toBe("Step 0");
  });

  it("follows the heading as the tour advances", async () => {
    const user = userEvent.setup();
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);

    await user.click(screen.getByText("onboarding.next"));
    expect(dialogName()).toBe("Step 1");

    await user.click(screen.getByText("onboarding.next"));
    expect(dialogName()).toBe("Step 2");
  });

  it("names the exit question after its own heading", async () => {
    const user = userEvent.setup();
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);

    await user.click(screen.getByText("onboarding.skip"));

    expect(dialogName()).toBe("onboarding.exit.title");
  });

  it("goes back to the step heading when the exit question is dismissed", async () => {
    const user = userEvent.setup();
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);

    await user.click(screen.getByText("onboarding.skip"));
    await user.click(screen.getByText("onboarding.exit.keepGoing"));

    expect(dialogName()).toBe("Step 0");
  });

  // A stale hard-coded label is exactly what this replaces.
  it("carries no static aria-label", () => {
    render(<WelcomeWizard open={true} onComplete={vi.fn()} />);

    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-label");
  });
});
