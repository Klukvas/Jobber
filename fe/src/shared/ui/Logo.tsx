import { useId } from "react";

interface LogoMarkProps {
  className?: string;
  /** Accessible name. Leave unset when the word "Cursopus" is rendered next to the mark. */
  label?: string;
}

/**
 * The Cursopus mark: a "C" drawn as an open path (cursus) with a waypoint dot
 * in its opening (opus, the destination). Fixed brand colours on purpose: the
 * lime badge keeps its contrast on both the light and the dark theme.
 */
export function LogoMark({ className = "h-7 w-7", label }: LogoMarkProps) {
  const gradientId = useId();
  return (
    <svg
      viewBox="0 0 32 32"
      className={className}
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      focusable="false"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#BEF264" />
          <stop offset="1" stopColor="#84CC16" />
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="9" fill={`url(#${gradientId})`} />
      <path
        d="M21.6 10.4A8.2 8.2 0 1 0 21.6 21.6"
        fill="none"
        stroke="#0B0D12"
        strokeWidth="3.3"
        strokeLinecap="round"
      />
      <circle cx="23.2" cy="16" r="2.5" fill="#0B0D12" />
    </svg>
  );
}
