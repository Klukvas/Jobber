/** Colored circle with a lucide icon (used by iconic template). */
export function SectionIcon({
  icon: Icon,
  color,
}: {
  readonly icon: React.ComponentType<{ className?: string }>;
  readonly color: string;
}) {
  return (
    <span
      className="mr-2 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-white"
      style={{ backgroundColor: color }}
    >
      <Icon className="h-3.5 w-3.5" />
    </span>
  );
}
