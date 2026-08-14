import type {
  JobDTO,
  MoveTarget,
  StagePhase,
  StageTemplateDTO,
} from "@/shared/types/api";

// A unified-board column is either a phase base column (always present for
// wishlist/applied/offer/rejected) or one of the user's stage columns.
// In Progress has no base column — being "in progress" means being on a stage.
export interface UnifiedColumn {
  id: string;
  kind: "phase" | "stage" | "archived";
  phase: StagePhase | "rejected" | "archived";
  template?: StageTemplateDTO;
  labelKey?: string; // for phase columns; stage columns use template.name
}

export const UNIFIED_PHASE_LABEL_KEYS: Record<string, string> = {
  wishlist: "stages.phase.wishlist",
  applied: "stages.phase.applied",
  offer: "stages.phase.offer",
  rejected: "jobs.board.rejected",
  archived: "jobs.board.archived",
};

function phaseColumnId(phase: string): string {
  return `phase:${phase}`;
}

function stageColumnId(template: StageTemplateDTO): string {
  return `stage:${template.id}`;
}

// Column layout: [Wishlist base][wishlist stages][Applied base][applied
// stages][in_progress stages][Offer base][offer stages][Rejected]
// (+[Archived] when the filter is on).
export function buildUnifiedColumns(
  templates: StageTemplateDTO[],
  showArchived: boolean,
): UnifiedColumn[] {
  const byPhase = (phase: StagePhase) =>
    templates
      .filter((tpl) => tpl.phase === phase)
      // All items share `phase` after the filter, so order alone decides.
      .sort((a, b) => a.order - b.order)
      .map((tpl): UnifiedColumn => ({
        id: stageColumnId(tpl),
        kind: "stage",
        phase: tpl.phase,
        template: tpl,
      }));

  const columns: UnifiedColumn[] = [
    {
      id: phaseColumnId("wishlist"),
      kind: "phase",
      phase: "wishlist",
      labelKey: UNIFIED_PHASE_LABEL_KEYS.wishlist,
    },
    ...byPhase("wishlist"),
    {
      id: phaseColumnId("applied"),
      kind: "phase",
      phase: "applied",
      labelKey: UNIFIED_PHASE_LABEL_KEYS.applied,
    },
    ...byPhase("applied"),
    ...byPhase("in_progress"),
    {
      id: phaseColumnId("offer"),
      kind: "phase",
      phase: "offer",
      labelKey: UNIFIED_PHASE_LABEL_KEYS.offer,
    },
    ...byPhase("offer"),
    {
      id: phaseColumnId("rejected"),
      kind: "phase",
      phase: "rejected",
      labelKey: UNIFIED_PHASE_LABEL_KEYS.rejected,
    },
    ...byPhase("rejected"),
  ];

  if (showArchived) {
    columns.push({
      id: phaseColumnId("archived"),
      kind: "archived",
      phase: "archived",
      labelKey: UNIFIED_PHASE_LABEL_KEYS.archived,
    });
  }
  return columns;
}

// Places a card into a column id. Status is authoritative for terminal
// zones; the current stage decides the position inside the active zone.
// Stage matching is by name (JobDTO carries current_stage_name; the
// current_stage_id is the job_stage instance, not the template).
export function placeJob(
  job: JobDTO,
  templates: StageTemplateDTO[],
): string | null {
  const currentTemplate = job.current_stage_name
    ? templates.find((tpl) => tpl.name === job.current_stage_name)
    : undefined;

  switch (job.status) {
    case "archived":
      return phaseColumnId("archived"); // rendered only when the filter is on
    case "rejected":
      return currentTemplate?.phase === "rejected"
        ? stageColumnId(currentTemplate)
        : phaseColumnId("rejected");
    case "saved":
      return currentTemplate?.phase === "wishlist"
        ? stageColumnId(currentTemplate)
        : phaseColumnId("wishlist");
    case "offer":
      return currentTemplate?.phase === "offer"
        ? stageColumnId(currentTemplate)
        : phaseColumnId("offer");
    default:
      // applied / on_hold — the active zone
      if (
        currentTemplate &&
        (currentTemplate.phase === "applied" ||
          currentTemplate.phase === "in_progress")
      ) {
        return stageColumnId(currentTemplate);
      }
      return phaseColumnId("applied");
  }
}

// Translates a drop on a column into the atomic move payload.
// Archived is not a move target (archive stays a menu action).
export function columnMoveTarget(column: UnifiedColumn): MoveTarget | null {
  if (column.kind === "stage" && column.template) {
    return { type: "stage", stage_template_id: column.template.id };
  }
  if (column.kind === "phase" && column.phase !== "archived") {
    return {
      type: "phase",
      phase: column.phase as StagePhase,
    };
  }
  return null;
}

// Legacy dedup: users whose old templates mirror base columns (an "Applied"
// stage in the applied phase) would see two identical columns. An EMPTY base
// column is hidden when its phase already has stage columns; it reappears as
// soon as a card lands in the "no stage" state. Archived never hides.
export function shouldHideBase(
  column: UnifiedColumn,
  jobCount: number,
  allColumns: UnifiedColumn[],
): boolean {
  if (column.kind !== "phase") return false;
  if (column.phase === "archived") return false;
  if (jobCount > 0) return false;
  return allColumns.some((c) => c.kind === "stage" && c.phase === column.phase);
}
