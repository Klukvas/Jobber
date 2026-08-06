/** Tailwind top-border color classes keyed by job status / column ID */
export const STATUS_TOP_BORDER_COLORS: Record<string, string> = {
  saved: "border-t-sky-500",
  applied: "border-t-green-500",
  on_hold: "border-t-yellow-500",
  offer: "border-t-blue-500",
  rejected: "border-t-red-500",
  archived: "border-t-gray-500",
  // Unified-board base columns
  "phase:wishlist": "border-t-sky-500",
  "phase:applied": "border-t-green-500",
  "phase:offer": "border-t-blue-500",
  "phase:rejected": "border-t-red-500",
  "phase:archived": "border-t-gray-500",
};

/** Tailwind left-border color classes keyed by job status / column ID */
export const STATUS_LEFT_BORDER_COLORS: Record<string, string> = {
  saved: "border-l-sky-500",
  applied: "border-l-green-500",
  on_hold: "border-l-yellow-500",
  offer: "border-l-blue-500",
  rejected: "border-l-red-500",
  archived: "border-l-gray-500",
};
