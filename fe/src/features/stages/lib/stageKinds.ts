import { STAGE_KINDS, type StageKind } from "@/shared/types/api";

/** The kind a column has when nobody has classified it (or the payload is old). */
export const DEFAULT_STAGE_KIND: StageKind = "other";

/**
 * The dropdown, in application-lifecycle order — the same order the backend
 * lists its kinds in, so the control reads like a pipeline rather than an
 * alphabetised enum. `other` stays last: it is the fallback, not a step.
 */
export const STAGE_KIND_OPTIONS: readonly StageKind[] = STAGE_KINDS;

/** Locale key for a kind's label, used by the dropdown and the stage list. */
export function stageKindLabelKey(kind: StageKind): string {
  return `stages.kinds.${kind}`;
}

/**
 * The kind of a template as the UI should treat it. A payload written before
 * kinds existed — or an unknown value from a newer backend — reads as "other",
 * which is the conservative choice: it claims no analytics meaning.
 */
export function stageKindOf(template: { kind?: string | null }): StageKind {
  const kind = template.kind;
  return STAGE_KINDS.includes(kind as StageKind)
    ? (kind as StageKind)
    : DEFAULT_STAGE_KIND;
}
