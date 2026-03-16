import { useRef, useEffect, type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/utils";

interface SidebarPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly title: string;
  readonly children: ReactNode;
  /** When true, renders as a full-screen modal (for mobile sheet context). */
  readonly fullscreen?: boolean;
}

export function SidebarPopover({
  isOpen,
  onClose,
  title,
  children,
  fullscreen = false,
}: SidebarPopoverProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    if (fullscreen) {
      const handleKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") onClose();
      };
      document.addEventListener("keydown", handleKey);
      return () => document.removeEventListener("keydown", handleKey);
    }

    function handleClick(e: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    // Delay to prevent immediate close from the icon click
    const timer = setTimeout(() => {
      document.addEventListener("mousedown", handleClick);
    }, 0);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("mousedown", handleClick);
    };
  }, [isOpen, onClose, fullscreen]);

  if (!isOpen) return null;

  if (fullscreen) {
    return (
      <div
        ref={backdropRef}
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
        onClick={(e) => {
          if (e.target === backdropRef.current) onClose();
        }}
      >
        <div
          ref={panelRef}
          role="dialog"
          aria-modal="true"
          className="relative mx-4 flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-xl bg-background shadow-2xl"
        >
          <div className="flex items-center justify-between border-b px-6 py-4">
            <h3 className="text-lg font-semibold">{title}</h3>
            <button
              onClick={onClose}
              className="flex h-8 w-8 items-center justify-center rounded-lg hover:bg-muted"
              aria-label="Close"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="overflow-y-auto p-6">{children}</div>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={panelRef}
      className={cn(
        "absolute left-14 top-0 z-40 h-full w-[280px] overflow-y-auto border-r bg-background shadow-lg",
        "animate-in slide-in-from-left-2 duration-200",
      )}
    >
      <div className="flex items-center justify-between border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{title}</h3>
        <button
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded hover:bg-muted"
          aria-label="Close"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
      <div className="p-4">{children}</div>
    </div>
  );
}
