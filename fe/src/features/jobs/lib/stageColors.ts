// Deterministic, semantic-ish colour for a pipeline stage.
//
// Terminal outcomes get fixed hues (archived → slate, rejected → red, offer →
// emerald); every other stage gets a stable colour hashed from its name, so the
// same stage is always the same colour and different stages stand apart. This
// lets you scan the board/list by colour instead of reading every pill.
//
// All class strings are literals (not built by interpolation) so Tailwind's JIT
// keeps them.

export interface StageColor {
  /** badge / pill classes */
  pill: string;
  /** left-accent border colour (pair with `border-l-4`) */
  border: string;
  /** small dot / indicator background */
  dot: string;
}

const PALETTE: StageColor[] = [
  { pill: "bg-blue-500/10 text-blue-600 dark:text-blue-400", border: "border-l-blue-500", dot: "bg-blue-500" },
  { pill: "bg-indigo-500/10 text-indigo-600 dark:text-indigo-400", border: "border-l-indigo-500", dot: "bg-indigo-500" },
  { pill: "bg-violet-500/10 text-violet-600 dark:text-violet-400", border: "border-l-violet-500", dot: "bg-violet-500" },
  { pill: "bg-cyan-500/10 text-cyan-600 dark:text-cyan-400", border: "border-l-cyan-500", dot: "bg-cyan-500" },
  { pill: "bg-amber-500/10 text-amber-600 dark:text-amber-400", border: "border-l-amber-500", dot: "bg-amber-500" },
  { pill: "bg-teal-500/10 text-teal-600 dark:text-teal-400", border: "border-l-teal-500", dot: "bg-teal-500" },
  { pill: "bg-fuchsia-500/10 text-fuchsia-600 dark:text-fuchsia-400", border: "border-l-fuchsia-500", dot: "bg-fuchsia-500" },
  { pill: "bg-sky-500/10 text-sky-600 dark:text-sky-400", border: "border-l-sky-500", dot: "bg-sky-500" },
];

const NEUTRAL: StageColor = { pill: "bg-slate-400/15 text-slate-500 dark:text-slate-400", border: "border-l-slate-400", dot: "bg-slate-400" };
const REJECTED: StageColor = { pill: "bg-red-500/10 text-red-600 dark:text-red-400", border: "border-l-red-500", dot: "bg-red-500" };
const OFFER: StageColor = { pill: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400", border: "border-l-emerald-500", dot: "bg-emerald-500" };

function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

/** Resolve the colour for a card's current stage. */
export function stageColor(
  stageName: string | null | undefined,
  isArchived?: boolean,
): StageColor {
  if (isArchived) return NEUTRAL;
  const name = (stageName ?? "").trim().toLowerCase();
  if (!name) return NEUTRAL;
  if (/reject|declin/.test(name)) return REJECTED;
  if (/offer|hired|accept/.test(name)) return OFFER;
  return PALETTE[hash(name) % PALETTE.length];
}
