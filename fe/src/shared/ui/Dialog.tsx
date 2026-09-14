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
import { useBodyScrollLock } from "@/shared/hooks/useBodyScrollLock";
import { useDialogFocus } from "@/shared/hooks/useDialogFocus";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { useTopmostEscape } from "@/shared/hooks/useTopmostEscape";

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
  /**
   * Id of the element that names this dialog, when that element is not this
   * dialog's own `DialogTitle`.
   *
   * Normally nothing needs to pass this: a `DialogTitle` rendered anywhere
   * inside registers itself and becomes the name (see `DialogTitleContext`).
   * The override exists for the auth modals, whose headings are addressed by a
   * fixed id shared with their own tests.
   */
  labelledBy?: string;
  /**
   * True while an overlay this dialog does not own is on screen — the billing
   * provider's payment popup, which its script appends to `<body>`, outside
   * this dialog's DOM.
   *
   * The dialog keeps rendering, but stops managing the keyboard: the Tab trap
   * would lock focus out of the payment form, and Escape would close the modal
   * out from under a payment in flight. Ownership returns the moment the
   * overlay does.
   */
  hasExternalOverlay?: boolean;
}

/**
 * How a dialog learns its own name.
 *
 * `role="dialog"` with no accessible name is announced as "dialog", and nothing
 * else: the user is told a modal opened and not which one. Every dialog in this
 * app already draws a heading, so rather than asking thirty call sites to wire
 * that heading up by hand — and silently losing the name at the ones that
 * forget — `DialogTitle` registers itself here and `Dialog` labels itself with
 * whatever registered.
 *
 * Registration rather than a shared generated id, because several dialogs swap
 * between headings (a loading branch, a not-connected branch, the form) and two
 * headings must never end up carrying the same id. Whichever is mounted is the
 * one that names the dialog; a dialog with no heading at all emits no
 * `aria-labelledby`, so the fault is a missing name rather than a reference
 * that points nowhere.
 *
 * The value is the registration function itself: it takes the heading's id and
 * returns its own undo.
 */
const DialogTitleContext = React.createContext<
  ((id: string) => () => void) | null
>(null);

// Release past this distance (px) or faster than this velocity (px/s) dismisses.
const SHEET_DISMISS_OFFSET = 120;
const SHEET_DISMISS_VELOCITY = 600;

export function Dialog({
  open,
  onOpenChange,
  children,
  className,
  labelledBy,
  swipeToDismiss = false,
  hasExternalOverlay = false,
}: DialogProps) {
  const dialogRef = React.useRef<HTMLDivElement>(null);

  // Ids of the headings currently mounted inside. Normally one; a dialog that
  // swaps branches has none for the instant between an unmount and a mount, and
  // the first is the one that names the dialog either way.
  const [titleIds, setTitleIds] = React.useState<readonly string[]>([]);
  const registerTitle = React.useCallback((id: string) => {
    setTitleIds((current) => [...current, id]);
    return () =>
      setTitleIds((current) => current.filter((entry) => entry !== id));
  }, []);
  const labelId = labelledBy ?? titleIds[0];

  const isSmallScreen = useMediaQuery("(max-width: 639px)");
  const prefersReducedMotion = useReducedMotion();
  const asSheet = swipeToDismiss && isSmallScreen && !prefersReducedMotion;

  const y = useMotionValue(0);
  const overlayOpacity = useTransform(y, [0, 500], [1, 0.15]);
  const dragControls = useDragControls();

  // Focus starts inside, stays inside, and goes back where it came from. The
  // trap suspends while an overlay this dialog does not own is up; the
  // open/close half does not, or handing the keyboard over and back would
  // snatch focus out of the payment form.
  useDialogFocus(dialogRef, { open, trapEnabled: !hasExternalOverlay });

  // Freeze the page behind. The lock is counted across every overlay on the
  // page, so a dialog closing under another one cannot hand the scrollbar back
  // while that one is still up.
  useBodyScrollLock(open);

  // Escape closes — but only while this dialog is the topmost thing on the
  // page. Two things can be above it. An overlay it does not own (the payment
  // popup) lives outside its DOM, and closing the page behind a payment in
  // flight would lose it. And another dialog can be open over this one: each
  // dialog used to hold its own `document` listener, so one Escape ran all of
  // them and a confirmation opened over a form took the form down with it.
  const closeDialog = React.useCallback(
    () => onOpenChange(false),
    [onOpenChange],
  );
  useTopmostEscape(open && !hasExternalOverlay, dialogRef, closeDialog);

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
      <DialogTitleContext.Provider value={registerTitle}>
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
            aria-labelledby={labelId}
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
              className="absolute left-1/2 top-[env(safe-area-inset-top)] z-10 -translate-x-1/2 cursor-grab touch-none px-6 py-3 active:cursor-grabbing"
            >
              <span className="block h-1.5 w-10 rounded-full bg-muted-foreground/30" />
            </div>
            {children}
          </motion.div>
        </div>
      </DialogTitleContext.Provider>
    );
  }

  return (
    <DialogTitleContext.Provider value={registerTitle}>
      <div
        className="fixed inset-0 z-50 flex items-center justify-center"
        onClick={() => onOpenChange(false)}
      >
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
        <div
          ref={dialogRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby={labelId}
          tabIndex={-1}
          className={boxClassName}
          onClick={(e) => e.stopPropagation()}
        >
          {children}
        </div>
      </div>
    </DialogTitleContext.Provider>
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
        "max-h-[90vh] overflow-y-auto overscroll-contain",
        className,
      )}
      {...props}
    >
      {onClose && (
        <button
          onClick={onClose}
          aria-label={t("common.close")}
          // The X on its own was a 16×16 target — half the 44px a thumb needs,
          // and the one control every modal must have. The button is a 44×44
          // box on phones and a compact 32×32 from `sm` up; both offsets are
          // chosen so the glyph itself lands exactly where it was drawn
          // before (2 + 22 and 8 + 16 both centre it 24px in), so nothing
          // moves — only the area that answers a tap grows.
          className="absolute right-0.5 top-[max(0.125rem,env(safe-area-inset-top))] flex h-11 w-11 items-center justify-center rounded-md opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none sm:right-2 sm:top-[max(0.5rem,env(safe-area-inset-top))] sm:h-8 sm:w-8"
        >
          <X className="h-4 w-4" aria-hidden />
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

/**
 * The dialog's visible heading — and, by registering itself with the enclosing
 * `Dialog`, its accessible name.
 *
 * A caller may still pass its own `id` (the auth modals do, so their tests can
 * address the heading by a fixed name); whatever id the heading ends up with is
 * the one that gets registered.
 */
export function DialogTitle({
  className,
  id,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  const registerTitle = React.useContext(DialogTitleContext);
  const generatedId = React.useId();
  const titleId = id ?? generatedId;

  // A layout effect, so the name is on the dialog before the browser paints
  // the frame in which focus moves into it.
  React.useLayoutEffect(() => {
    if (!registerTitle) return;
    return registerTitle(titleId);
  }, [registerTitle, titleId]);

  return (
    <h2
      id={titleId}
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
