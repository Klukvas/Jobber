import { useState, useCallback, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useSidebarStore } from "@/stores/sidebarStore";
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
};

interface WelcomeWizardProps {
  open: boolean;
  onComplete: () => void;
}

export function WelcomeWizard({ open, onComplete }: WelcomeWizardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isExpanded = useSidebarStore((s) => s.isExpanded);
  const [currentStep, setCurrentStep] = useState(0);
  const dialogRef = useRef<HTMLDivElement>(null);

  const isFirst = currentStep === 0;
  const isLast = currentStep === TOTAL_STEPS - 1;

  // Sync highlight with current step
  useEffect(() => {
    if (open) {
      setOnboardingHighlight(STEP_HIGHLIGHT[currentStep] ?? null);
    }
    return () => setOnboardingHighlight(null);
  }, [currentStep, open]);

  // Lock body scroll only when open; restore previous value on cleanup
  useEffect(() => {
    if (!open) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open]);

  // Focus the dialog when it opens
  useEffect(() => {
    if (open) {
      dialogRef.current?.focus();
    }
  }, [open]);

  // Escape key to skip
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onComplete();
      }
    };
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, onComplete]);

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

  const handleSkip = useCallback(() => {
    onComplete();
  }, [onComplete]);

  if (!open) return null;

  // Sidebar width: 256px (w-64) expanded, 64px (w-16) collapsed
  const sidebarWidth = isExpanded ? 256 : 64;

  return (
    <>
      {/* Backdrop — only covers the content area, not the sidebar */}
      <div
        className="fixed inset-0 z-40 hidden bg-black/50 md:block"
        style={{ left: sidebarWidth }}
        onClick={handleSkip}
      />
      {/* Mobile: full overlay (sidebar is hidden on mobile) */}
      <div
        className="fixed inset-0 z-40 bg-black/50 md:hidden"
        onClick={handleSkip}
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
          aria-label={t("onboarding.welcome.title")}
          tabIndex={-1}
          className="relative m-4 w-full max-w-md rounded-lg border bg-background p-6 shadow-lg pointer-events-auto outline-none"
          onClick={(e) => e.stopPropagation()}
        >
          <WizardStepContent step={currentStep} />

          <div className="flex items-center justify-between pt-2">
            <div className="w-20">
              {!isFirst && (
                <Button variant="ghost" size="sm" onClick={handleBack}>
                  {t("onboarding.back")}
                </Button>
              )}
            </div>

            <StepIndicator currentStep={currentStep} totalSteps={TOTAL_STEPS} />

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
        </div>
      </div>
    </>
  );
}
