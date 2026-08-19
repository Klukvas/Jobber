// A small colored monogram for a company — deterministic colour + initials from
// the name, so each company gets a stable visual identity across the app instead
// of the same generic building icon everywhere. Class strings are literals so
// Tailwind's JIT keeps them.

interface CompanyAvatarProps {
  name?: string | null;
  size?: "sm" | "md";
  className?: string;
}

const COLORS = [
  "bg-blue-500",
  "bg-indigo-500",
  "bg-violet-500",
  "bg-fuchsia-500",
  "bg-rose-500",
  "bg-amber-500",
  "bg-teal-500",
  "bg-cyan-500",
  "bg-emerald-500",
];

function hash(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

function initials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return "?";
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}

export function CompanyAvatar({ name, size = "md", className = "" }: CompanyAvatarProps) {
  const label = (name ?? "").trim();
  const color = label ? COLORS[hash(label) % COLORS.length] : "bg-muted-foreground/40";
  const dims = size === "sm" ? "h-5 w-5 text-[9px]" : "h-8 w-8 text-xs";

  return (
    <span
      aria-hidden
      className={`inline-flex ${dims} shrink-0 items-center justify-center rounded-md font-semibold text-white ${color} ${className}`}
    >
      {label ? initials(label) : "?"}
    </span>
  );
}
