import * as React from "react";
import { useTranslation } from "react-i18next";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/utils";
import { useBodyScrollLock } from "@/shared/hooks/useBodyScrollLock";
import { useDialogFocus } from "@/shared/hooks/useDialogFocus";
import { useTopmostEscape } from "@/shared/hooks/useTopmostEscape";

interface SheetProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
  readonly title?: string;
  readonly children: React.ReactNode;
  readonly className?: string;
}

/**
 * Mobile bottom sheet / drawer.
 * Slides up from the bottom, takes ~85vh height.
 */
export function Sheet({
  open,
  onOpenChange,
  title,
  children,
  className,
}: SheetProps) {
  const { t } = useTranslation();
  const sheetRef = React.useRef<HTMLDivElement>(null);
  // Per instance. A fixed "sheet-heading" put the same id on every sheet on the
  // page, so two open sheets both pointed `aria-labelledby` at whichever
  // heading the document happened to hold first — one sheet announced with the
  // other's title.
  const headingId = React.useId();

  // Same containment as Dialog: focus lands inside on open, Tab cannot walk
  // out to the page behind, and the opener gets focus back on close.
  useDialogFocus(sheetRef, { open });

  // Escape closes this sheet only while it is the topmost overlay — the same
  // stack the dialogs use, so a sheet and a dialog over it cannot both answer
  // one keypress.
  const closeSheet = React.useCallback(
    () => onOpenChange(false),
    [onOpenChange],
  );
  useTopmostEscape(open, sheetRef, closeSheet);

  // Counted across every overlay on the page — see useBodyScrollLock.
  useBodyScrollLock(open);

  if (!open) return null;

  return (
    <div
      className={cn("fixed inset-0 z-50", className)}
      onClick={() => onOpenChange(false)}
    >
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50" />

      {/* Sheet */}
      <div
        ref={sheetRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? headingId : undefined}
        tabIndex={-1}
        className="fixed inset-x-0 bottom-0 z-50 flex max-h-[85vh] flex-col rounded-t-xl bg-background shadow-lg outline-none"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header. The sheet only ever renders on a touch layout, so the close
            button is a full 44x44 box rather than the 24x24 that `p-1` around a
            16px glyph gave it. The row's own padding shrinks by the same amount,
            so the header keeps the 48px height it had. */}
        <div className="flex items-center justify-between border-b px-4 py-0.5">
          <h2 id={headingId} className="text-sm font-semibold">
            {title}
          </h2>
          <button
            onClick={() => onOpenChange(false)}
            aria-label={t("common.close")}
            className="-mr-2 flex h-11 w-11 items-center justify-center rounded-sm opacity-70 hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">{children}</div>
      </div>
    </div>
  );
}
