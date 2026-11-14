import { useLayoutEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface MovablePanelProps {
  readonly node: HTMLElement;
  readonly children: ReactNode;
}

/**
 * Renders its children once into `node` (from `useMovablePanelNode`). Render it
 * outside any branch that depends on the viewport.
 */
export function MovablePanel({ node, children }: MovablePanelProps) {
  return createPortal(children, node);
}

interface MovablePanelOutletProps {
  readonly node: HTMLElement;
  readonly className?: string;
}

/** Where the panel currently shows. Mount at most one outlet at a time. */
export function MovablePanelOutlet({
  node,
  className,
}: MovablePanelOutletProps) {
  const hostRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    host.appendChild(node);
    return () => {
      if (node.parentNode === host) host.removeChild(node);
    };
  }, [node]);

  return <div ref={hostRef} className={className} />;
}
