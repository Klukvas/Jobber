import { useState, useRef, type ReactNode } from "react";
import { cn } from "@/shared/lib/utils";

type TooltipSide = "top" | "right" | "bottom" | "left";

interface TooltipProps {
  readonly content: string;
  readonly side?: TooltipSide;
  readonly delayMs?: number;
  readonly children: ReactNode;
  readonly className?: string;
}

const sideClasses: Record<TooltipSide, string> = {
  top: "bottom-full left-1/2 -translate-x-1/2 mb-2",
  bottom: "top-full left-1/2 -translate-x-1/2 mt-2",
  left: "right-full top-1/2 -translate-y-1/2 mr-2",
  right: "left-full top-1/2 -translate-y-1/2 ml-2",
};

const arrowClasses: Record<TooltipSide, string> = {
  top: "top-full left-1/2 -translate-x-1/2 border-t-gray-900 border-x-transparent border-b-transparent border-4",
  bottom:
    "bottom-full left-1/2 -translate-x-1/2 border-b-gray-900 border-x-transparent border-t-transparent border-4",
  left: "left-full top-1/2 -translate-y-1/2 border-l-gray-900 border-y-transparent border-r-transparent border-4",
  right:
    "right-full top-1/2 -translate-y-1/2 border-r-gray-900 border-y-transparent border-l-transparent border-4",
};

export function Tooltip({
  content,
  side = "right",
  delayMs = 300,
  children,
  className,
}: TooltipProps) {
  const [visible, setVisible] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const show = () => {
    timerRef.current = setTimeout(() => setVisible(true), delayMs);
  };

  const hide = () => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setVisible(false);
  };

  return (
    <div
      className={cn("relative inline-flex", className)}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {visible && (
        <div
          className={cn(
            "pointer-events-none absolute z-50 whitespace-nowrap rounded-md bg-gray-900 px-2.5 py-1.5 text-xs text-white shadow-lg",
            "animate-in fade-in-0 zoom-in-95 duration-150",
            sideClasses[side],
          )}
          role="tooltip"
        >
          {content}
          <span className={cn("absolute block h-0 w-0", arrowClasses[side])} />
        </div>
      )}
    </div>
  );
}
