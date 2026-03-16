/** Maps skill level strings to numeric values for visual renderers. */
export const SKILL_LEVEL_MAP: Record<string, number> = {
  beginner: 1,
  intermediate: 2,
  advanced: 3,
  expert: 4,
  master: 5,
};

/** Maximum numeric skill level value (master = 5). */
export const MAX_SKILL_LEVEL = 5;

/** Total number of dots rendered in the dots display mode. */
export const MAX_DOTS = 5;

/** Converts a skill level string to a number (0 if unknown/empty). */
export function skillLevelToNumber(level: string): number {
  return SKILL_LEVEL_MAP[level.toLowerCase()] ?? 0;
}
