import * as React from "react";

import { isRendered } from "@/shared/lib/visibility";

/**
 * What Tab can reach. `[tabindex="-1"]` is deliberately excluded: the dialog
 * container itself carries it so it can be focused programmatically, and it
 * must not count as a stop in the tab order.
 */
const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(", ");

/**
 * Everything inside `container` that a keyboard can land on, in tab order.
 *
 * Hidden controls are left out, and that is not a refinement — it is what makes
 * the wrap-around work. Dialogs here routinely keep controls mounted and hide
 * them: a save button that appears once a field changes, a wizard step that is
 * not the current one. Counting those made the "last" element of this list
 * something with no box on screen, `focus()` on it did nothing at all, and the
 * keyboard stayed wherever it was — one Tab from walking out of the modal
 * entirely.
 */
export function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
  ).filter(
    (element) =>
      !element.hasAttribute("disabled") &&
      element.getAttribute("aria-hidden") !== "true" &&
      isRendered(element),
  );
}

/**
 * How many frames the open-focus retry will wait for a container that is still
 * arriving on screen. The slowest of them is the navigation drawer's 300ms
 * transition — around eighteen frames at 60Hz — and the retry stops the moment
 * focus lands, so this is only the ceiling for a panel that never becomes
 * focusable at all.
 */
const FOCUS_RETRY_FRAMES = 20;

interface DialogFocusOptions {
  readonly open: boolean;
  /**
   * False while something other than this dialog should own the keyboard —
   * an overlay outside its DOM, so trapping Tab would lock focus out of it.
   */
  readonly trapEnabled?: boolean;
}

/**
 * Keyboard containment for a modal: focus moves inside when it opens, cannot
 * leave while it is up, and returns to whatever opened it when it closes.
 *
 * Two things were wrong before, and they compounded. Focus was moved inside
 * from a `requestAnimationFrame` callback, which is a frame the dialog does not
 * control — the auth modal opened with `document.activeElement` still on
 * `<body>`. And the trap only acted when focus was already sitting on the first
 * or last control inside, so from anywhere else — `<body>` included — a real
 * Tab walked straight into the navbar behind the modal. The container is
 * focused synchronously in a layout effect now, and Tab from outside is pulled
 * back in rather than ignored.
 */
export function useDialogFocus(
  containerRef: React.RefObject<HTMLElement | null>,
  { open, trapEnabled = true }: DialogFocusOptions,
): void {
  const previousFocusRef = React.useRef<HTMLElement | null>(null);

  /**
   * Hands the keyboard back to whatever opened the overlay — but only if it is
   * still on the page. A route-driven modal (the auth modals) can leave its
   * trigger unmounted, and focusing a detached node silently drops focus on
   * `<body>`, stranding keyboard users at the top of the document.
   */
  const restorePreviousFocus = React.useCallback(() => {
    const trigger = previousFocusRef.current;
    if (trigger?.isConnected) trigger.focus();
  }, []);

  // Opening and closing. Kept apart from the keyboard effect below, which
  // suspends while an external overlay is up — re-running *this* one on that
  // switch would snatch focus back out of the overlay.
  React.useLayoutEffect(() => {
    if (!open) {
      // The cleanup below already asked for this focus move, and for an
      // overlay that unmounts as it closes that is the end of it. For one that
      // stays mounted — the app's navigation drawer, which is only pushed off
      // screen — it is not: a layout effect's *cleanup* runs inside React's
      // mutation phase, and React restores the selection it captured before
      // that phase afterwards. The container was what had focus, the container
      // is still in the document, so React put the keyboard straight back into
      // the panel the visitor had just dismissed. This runs after that restore.
      restorePreviousFocus();
      previousFocusRef.current = null;
      return;
    }

    previousFocusRef.current = document.activeElement as HTMLElement | null;
    moveFocusInside(containerRef.current);

    // A container that is not on screen yet cannot be focused, and an overlay
    // is exactly that for the first frames after it opens. The drawer's
    // `transition-all` covers `visibility`, and a transition's first sample is
    // still the value it started from — so the panel computes to
    // `visibility: hidden` while it is already sliding in, refusing `focus()`,
    // with nothing inside it rendered to fall back to either. Two attempts, the
    // layout effect and one frame later, could both land on that sample: no
    // `focusin` fired at all, and the keyboard stayed on the hamburger
    // underneath the panel it had just opened.
    //
    // So retry until the container will actually take focus rather than a fixed
    // number of times. Nothing is stolen while waiting — an unfocusable
    // container is one `moveFocusInside` cannot move anything into — and the
    // first attempt that lands ends the loop, so focus a control inside has
    // since been given is left alone.
    let framesLeft = FOCUS_RETRY_FRAMES;
    let frame = 0;
    const focusOnceFocusable = () => {
      const container = containerRef.current;
      if (!container || container.contains(document.activeElement)) return;

      moveFocusInside(container);
      if (container.contains(document.activeElement)) return;

      framesLeft -= 1;
      if (framesLeft > 0) frame = requestAnimationFrame(focusOnceFocusable);
    };
    frame = requestAnimationFrame(focusOnceFocusable);

    return () => {
      cancelAnimationFrame(frame);
      // Unmounting is the case this still has to cover on its own: the branch
      // above never runs for a component that is going away.
      restorePreviousFocus();
    };
  }, [open, containerRef, restorePreviousFocus]);

  React.useEffect(() => {
    if (!open || !trapEnabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Tab") return;
      const container = containerRef.current;
      if (!container) return;

      const focusable = getFocusableElements(container);
      const active = document.activeElement as HTMLElement | null;

      // Nothing to tab between: keep focus on the dialog itself rather than
      // letting it escape to the page underneath.
      if (focusable.length === 0) {
        event.preventDefault();
        container.focus();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const isInside = !!active && container.contains(active);

      // Focus that starts outside — on <body>, or on the page behind — is the
      // case the old trap missed entirely.
      if (!isInside) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
        return;
      }

      // The container holds focus on open and is not itself a tab stop, so
      // both directions have to be steered off it explicitly.
      if (event.shiftKey && (active === first || active === container)) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && (active === last || active === container)) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, trapEnabled, containerRef]);
}

/**
 * Puts focus on the dialog container, so a screen reader announces the dialog
 * rather than one control out of it. A control inside that already has focus —
 * an `autoFocus` field, say — keeps it.
 */
function moveFocusInside(container: HTMLElement | null): void {
  if (!container) return;
  if (container.contains(document.activeElement)) return;

  container.focus();
  if (document.activeElement === container) return;

  // The container refused focus (no tabindex, or not yet focusable): fall back
  // to the first control inside so the modal is never left with focus on body.
  getFocusableElements(container)[0]?.focus();
}
