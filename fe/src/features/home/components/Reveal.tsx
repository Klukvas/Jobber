import type { CSSProperties, ReactNode } from "react";
import { useInView } from "@/shared/hooks/useInView";
import { cn } from "@/shared/lib/utils";

interface RevealProps {
  readonly children: ReactNode;
  /** Extra transition delay for staggering sibling reveals. */
  readonly delayMs?: number;
  readonly className?: string;
}

/** Fades content up once it scrolls into view (landing sections only). */
export function Reveal({ children, delayMs = 0, className }: RevealProps) {
  const { ref, isInView } = useInView<HTMLDivElement>();

  return (
    <div
      ref={ref}
      className={cn("landing-reveal", isInView && "is-visible", className)}
      style={{ "--reveal-delay": `${delayMs}ms` } as CSSProperties}
    >
      {children}
    </div>
  );
}
