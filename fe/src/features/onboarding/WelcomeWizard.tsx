import { useState, useCallback, useEffect, useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useBodyScrollLock } from "@/shared/hooks/useBodyScrollLock";
import { useDialogFocus } from "@/shared/hooks/useDialogFocus";
import { Button } from "@/shared/ui/Button";
import { StepIndicator } from "./StepIndicator";
import { WizardStepContent, TOTAL_STEPS } from "./WizardStepContent";
import { setOnboardingHighlight } from "./useOnboarding";

/** Maps wizard step index -> sidebar path to highlight */
const STEP_HIGHLIGHT: Record<number, string | null> = {
  0: null,
  1: "/app/companies",
  2: "/app/resumes",
  3: "/app/jobs",
  4: "/app/stages",
  5: "/app/analytics",
  6: null,
  7: null,
};

interface WelcomeWizardProps {
  open: boolean;
  onComplete: () => void;
  /**
   * "Not now" — hide the tour until the next visit. Optional so the wizard
   * still works standalone; without it, leaving falls back to completing.
   */
  onDismissForSession?: () => void;
}

export function WelcomeWizard({
  open,
  onComplete,
  onDismissForSession,
}: WelcomeWizardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isExpanded = useSidebarStore((s) => s.isExpanded);
  const [currentStep, setCurrentStep] = useState(0);
  // Leaving the tour is a decision with two very different outcomes, so it is
  // asked rather than inferred from a keystroke or a stray backdrop click.
  const [isConfirmingExit, setIsConfirmingExit] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);
  const exitPanelRef = useRef<HTMLDivElement>(null);
  // The tour is one dialog showing eight different things. Naming it after
  // whichever heading is drawn keeps the spoken name and the visible one the
  // same — it used to say "Welcome to Jobber" on every step, including the
  // question about leaving.
  const stepHeadingId = useId();
  const exitHeadingId = useId();

  const isFirst = currentStep === 0;
  const isLast = currentStep === TOTAL_STEPS - 1;

  // Sync highlight with current step
  useEffect(() => {
    if (open) {
      setOnboardingHighlight(STEP_HIGHLIGHT[currentStep] ?? null);
    }
    return () => setOnboardingHighlight(null);
  }, [currentStep, open]);

  // Lock body scroll only when open. The lock is counted across every overlay
  // on the page, so the page stays still while any of them is up.
  useBodyScrollLock(open);

  // Focus lands on the card when the tour opens and cannot Tab out to the app
  // behind it — the tour covers the page it is describing, so a keyboard user
  // walking into the navbar underneath has no way of knowing where they went.
  useDialogFocus(dialogRef, { open });

  // Escape asks how to leave; a second Escape backs out of that question —
  // the least destructive reading of "I didn't mean to do that".
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setIsConfirmingExit((confirming) => !confirming);
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open]);

  // Move focus into the exit question when it appears, and hand it back to the
  // tour when it is dismissed, so a keyboard user is never left nowhere.
  useEffect(() => {
    if (!open) return;
    if (isConfirmingExit) {
      exitPanelRef.current?.focus();
    } else {
      dialogRef.current?.focus();
    }
  }, [open, isConfirmingExit]);

  const handleNext = useCallback(() => {
    if (isLast) {
      onComplete();
      navigate("/app/companies");
      return;
    }
    setCurrentStep((s) => s + 1);
  }, [isLast, onComplete, navigate]);

  const handleBack = useCallback(() => {
    setCurrentStep((s) => Math.max(0, s - 1));
  }, []);

  const askHowToLeave = useCallback(() => setIsConfirmingExit(true), []);
  const keepGoing = useCallback(() => setIsConfirmingExit(false), []);

  const dismissForNow = useCallback(() => {
    setIsConfirmingExit(false);
    (onDismissForSession ?? onComplete)();
  }, [onDismissForSession, onComplete]);

  const neverShowAgain = useCallback(() => {
    setIsConfirmingExit(false);
    onComplete();
  }, [onComplete]);

  if (!open) return null;

  // Sidebar width: 256px (w-64) expanded, 64px (w-16) collapsed
  const sidebarWidth = isExpanded ? 256 : 64;

  return (
    <>
      {/* Backdrop — only covers the content area, not the sidebar. Clicking it
          asks how to leave; it never decides on the customer's behalf. */}
      <div
        className="fixed inset-0 z-40 hidden bg-black/50 md:block"
        style={{ left: sidebarWidth }}
        onClick={askHowToLeave}
      />
      {/* Mobile: full overlay (sidebar is hidden on mobile) */}
      <div
        className="fixed inset-0 z-40 bg-black/50 md:hidden"
        onClick={askHowToLeave}
      />

      {/* Dialog card — offset by sidebar width on desktop, no offset on mobile */}
      <div
        className="fixed inset-0 z-50 flex items-center justify-center pointer-events-none max-md:!pl-0"
        style={{ paddingLeft: sidebarWidth }}
      >
        <div
          ref={dialogRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby={isConfirmingExit ? exitHeadingId : stepHeadingId}
          tabIndex={-1}
          className="relative m-4 w-full max-w-md rounded-lg border bg-background p-6 shadow-lg pointer-events-auto outline-none"
          onClick={(e) => e.stopPropagation()}
        >
          {isConfirmingExit ? (
            <div
              ref={exitPanelRef}
              role="group"
              aria-labelledby={exitHeadingId}
              tabIndex={-1}
              className="outline-none"
            >
              <h2 id={exitHeadingId} className="text-lg font-semibold">
                {t("onboarding.exit.title")}
              </h2>
              <p className="mt-2 text-sm text-muted-foreground">
                {t("onboarding.exit.description")}
              </p>
              <div className="mt-6 flex flex-col gap-2">
                <Button size="sm" onClick={keepGoing} autoFocus>
                  {t("onboarding.exit.keepGoing")}
                </Button>
                <Button size="sm" variant="outline" onClick={dismissForNow}>
                  {t("onboarding.exit.notNow")}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={neverShowAgain}
                  className="text-muted-foreground"
                >
                  {t("onboarding.exit.neverAgain")}
                </Button>
              </div>
              <p className="mt-4 text-xs text-muted-foreground">
                {t("onboarding.exit.restartHint")}
              </p>
            </div>
          ) : (
            <>
              <WizardStepContent step={currentStep} headingId={stepHeadingId} />

              <div className="flex items-center justify-between pt-2">
                <div className="w-20">
                  {!isFirst ? (
                    <Button variant="ghost" size="sm" onClick={handleBack}>
                      {t("onboarding.back")}
                    </Button>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={askHowToLeave}
                      className="text-muted-foreground"
                    >
                      {t("onboarding.skip")}
                    </Button>
                  )}
                </div>

                <StepIndicator
                  currentStep={currentStep}
                  totalSteps={TOTAL_STEPS}
                />

                <div className="flex w-20 justify-end">
                  {isLast ? (
                    <Button size="sm" onClick={handleNext}>
                      {t("onboarding.letsGo")}
                    </Button>
                  ) : (
                    <Button size="sm" onClick={handleNext}>
                      {t("onboarding.next")}
                    </Button>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </>
  );
}
