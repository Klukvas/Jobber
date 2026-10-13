/**
 * Stage template `order` values are 0-based on the API (the backend seeds a new
 * account's pipeline as 0..6). The Stages page shows positions 1-based, but that
 * is display only — anything created must continue after the highest order that
 * already exists, or it collides with the seeded columns.
 */
export function nextStageOrder(existing: { order: number }[]): number {
  if (existing.length === 0) return 0;
  return Math.max(...existing.map((s) => s.order)) + 1;
}

/** Orders for `count` new stages appended in sequence after `existing`. */
export function sequentialStageOrders(
  existing: { order: number }[],
  count: number,
): number[] {
  const start = nextStageOrder(existing);
  return Array.from({ length: count }, (_, i) => start + i);
}
