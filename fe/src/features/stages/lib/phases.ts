import type { StagePhase, StageTemplateDTO } from "@/shared/types/api";
import { STAGE_PHASES } from "@/shared/types/api";

export const PHASE_LABEL_KEYS: Record<StagePhase, string> = {
  wishlist: "stages.phase.wishlist",
  applied: "stages.phase.applied",
  in_progress: "stages.phase.inProgress",
  offer: "stages.phase.offer",
};

const PHASE_RANK: Record<StagePhase, number> = {
  wishlist: 0,
  applied: 1,
  in_progress: 2,
  offer: 3,
};

export function phaseRank(phase: StagePhase): number {
  return PHASE_RANK[phase] ?? PHASE_RANK.in_progress;
}

// Composite pipeline sort: phase first, template order within the phase
export function compareTemplates(
  a: StageTemplateDTO,
  b: StageTemplateDTO,
): number {
  const byPhase = phaseRank(a.phase) - phaseRank(b.phase);
  return byPhase !== 0 ? byPhase : a.order - b.order;
}

// Groups templates by phase preserving pipeline order. Empty phases are
// kept — the Stages page always shows all four groups so the pipeline
// structure stays visible.
export function groupTemplatesByPhase(
  templates: StageTemplateDTO[],
): { phase: StagePhase; templates: StageTemplateDTO[] }[] {
  return STAGE_PHASES.map((phase) => ({
    phase,
    templates: templates
      .filter((tpl) => tpl.phase === phase)
      .sort((a, b) => a.order - b.order),
  }));
}
