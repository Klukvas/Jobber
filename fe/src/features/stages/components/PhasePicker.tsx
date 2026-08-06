import { useTranslation } from "react-i18next";
import { STAGE_PHASES, type StagePhase } from "@/shared/types/api";
import { PHASE_LABEL_KEYS } from "../lib/phases";
import { cn } from "@/shared/lib/utils";

interface PhasePickerProps {
  value: StagePhase;
  onChange: (phase: StagePhase) => void;
}

// Segmented phase control. Pre-selected with in_progress by the callers, so
// users who don't think in phases never have to touch it.
export function PhasePicker({ value, onChange }: PhasePickerProps) {
  const { t } = useTranslation();

  return (
    <div
      role="radiogroup"
      aria-label={t("stages.phase.label")}
      className="flex flex-wrap gap-1 rounded-md bg-muted p-1"
    >
      {STAGE_PHASES.map((phase) => (
        <button
          key={phase}
          type="button"
          role="radio"
          aria-checked={value === phase}
          onClick={() => onChange(phase)}
          className={cn(
            "rounded px-2.5 py-1.5 text-xs font-medium transition-colors",
            value === phase
              ? "bg-background shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {t(PHASE_LABEL_KEYS[phase])}
        </button>
      ))}
    </div>
  );
}
