import * as React from "react";
import { useTranslation } from "react-i18next";
import { X } from "lucide-react";
import {
  motion,
  useMotionValue,
  useTransform,
  useDragControls,
  useReducedMotion,
  animate,
  type PanInfo,
} from "motion/react";
import { cn } from "@/shared/lib/utils";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";

interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
  className?: string;
  /**
   * On small screens, present the dialog as a sheet that materializes upward
   * and can be swiped down to dismiss. Ignored on >= sm and when the user
   * prefers reduced motion — both fall back to the standard centered dialog.
   */
  swipeToDismiss?: boolean;
}

// Release past this distance (px) or faster than this velocity (px/s) dismisses.
const SHEET_DISMISS_OFFSET = 120;
const SHEET_DISMISS_VELOCITY = 600;

export function Dialog({
  open,
  onOpenChange,
  children,
  className,
  swipeToDismiss = false,
}: DialogProps) {
  const dialogRef = React.useRef<HTMLDivElement>(null);
  const previousFocusRef = React.useRef<HTMLElement | null>(null);

  const isSmallScreen = useMediaQuery("(max-width: 639px)");
  const prefersReducedMotion = useReducedMotion();
  const asSheet = swipeToDismiss && isSmallScreen && !prefersReducedMotion;

  const y = useMotionValue(0);
  const overlayOpacity = useTransform(y, [0, 500], [1, 0.15]);
  const dragControls = useDragControls();

  React.useEffect(() => {
    if (!open) return;

    // Save currently focused element to restore later
    previousFocusRef.current = document.activeElement as HTMLElement;

    // Save previous overflow to restore on close (fixes nested dialogs)
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    // Focus the dialog container
    requestAnimationFrame(() => {
      dialogRef.current?.focus();
    });

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onOpenChange(false);
        return;
      }

      // Focus trap: constrain Tab within dialog
      if (e.key === "Tab" && dialogRef.current) {
        const focusable = dialogRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
        );
        if (focusable.length === 0) return;

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey) {
          if (document.activeElement === first) {
            e.preventDefault();
            last.focus();
          }
        } else {
          if (document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = previousOverflow;
      // Restore focus to the element that opened the dialog
      previousFocusRef.current?.focus();
    };
  }, [open, onOpenChange]);

  // Materialize the sheet upward each time it opens.
  React.useLayoutEffect(() => {
    if (!open || !asSheet) return;
    y.set(window.innerHeight);
    const controls = animate(y, 0, {
      type: "spring",
      bounce: 0,
      duration: 0.45,
    });
    return () => controls.stop();
  }, [open, asSheet, y]);

  if (!open) return null;

  const boxClassName = cn(
    "relative z-50 mx-auto w-full max-w-lg outline-none",
    className,
  );

  if (asSheet) {
    const handleDragEnd = (
      _event: MouseEvent | TouchEvent | PointerEvent,
      info: PanInfo,
    ) => {
      const flungDown =
        info.offset.y > SHEET_DISMISS_OFFSET ||
        info.velocity.y > SHEET_DISMISS_VELOCITY;
      if (flungDown) {
        // Hand the finger's velocity off to the closing spring — no seam
        // between the drag and the animation.
        animate(y, window.innerHeight, {
          type: "spring",
          bounce: 0,
          duration: 0.35,
          velocity: info.velocity.y,
        }).then(() => onOpenChange(false));
      } else {
        // Snap home, carrying velocity so a reversal has no brick wall.
        animate(y, 0, {
          type: "spring",
          bounce: 0.15,
          duration: 0.4,
          velocity: info.velocity.y,
        });
      }
    };

    return (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center"
        onClick={() => onOpenChange(false)}
      >
        <motion.div
          className="fixed inset-0 bg-black/50 backdrop-blur-sm"
          style={{ opacity: overlayOpacity }}
        />
        <motion.div
          ref={dialogRef}
          role="dialog"
          aria-modal="true"
          tabIndex={-1}
          className={boxClassName}
          style={{ y }}
          drag="y"
          dragListener={false}
          dragControls={dragControls}
          dragConstraints={{ top: 0 }}
          dragElastic={0.1}
          onDragEnd={handleDragEnd}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Grabber: the only drag origin, so inputs stay tappable. */}
          <div
            aria-hidden
            onPointerDown={(e) => dragControls.start(e)}
            className="absolute left-1/2 top-0 z-10 -translate-x-1/2 cursor-grab touch-none px-6 py-3 active:cursor-grabbing"
          >
            <span className="block h-1.5 w-10 rounded-full bg-muted-foreground/30" />
          </div>
          {children}
        </motion.div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      onClick={() => onOpenChange(false)}
    >
      <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        tabIndex={-1}
        className={boxClassName}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );
}

export function DialogContent({
  className,
  children,
  onClose,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { onClose?: () => void }) {
  const { t } = useTranslation();
  return (
    <div
      className={cn(
        "relative m-4 max-w-lg rounded-lg border bg-background p-6 shadow-lg",
        "max-h-[90vh] overflow-y-auto",
        className,
      )}
      {...props}
    >
      {onClose && (
        <button
          onClick={onClose}
          aria-label={t("common.close")}
          className="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none"
        >
          <X className="h-4 w-4" />
        </button>
      )}
      {children}
    </div>
  );
}

export function DialogHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col space-y-1.5 text-center sm:text-left",
        className,
      )}
      {...props}
    />
  );
}

export function DialogTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn(
        "text-lg font-semibold leading-none tracking-tight",
        className,
      )}
      {...props}
    />
  );
}

export function DialogDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={cn("text-sm text-muted-foreground", className)} {...props} />
  );
}

export function DialogFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end sm:space-x-2",
        className,
      )}
      {...props}
    />
  );
}
