/**
 * Keyboard safety while the billing provider's checkout covers the page.
 *
 * FastSpring's popup is a cross-origin iframe its script appends to `<body>`,
 * outside everything this app renders. Nothing about that appending moves the
 * keyboard: focus stayed on whatever app control had it, Tab walked through
 * buttons that were painted over by the checkout, and the payment form the
 * screen was showing could not be reached from the keyboard at all.
 *
 * So the page hands the keyboard over explicitly. Every top-level element that
 * is not part of the overlay is made `inert` and `aria-hidden` — one attribute
 * pair, applied to the app's own root rather than to the provider's DOM, which
 * this module never modifies — and focus is moved to the checkout frame. The
 * frame's contents stay entirely the provider's business: `focus()` on the
 * element is the whole interaction, and nothing here reads across the origin.
 *
 * Everything is put back when the overlay goes, including the focus the page
 * had when the checkout started.
 */

import { isRendered } from "@/shared/lib/visibility";

/** What the page looked like before the overlay took the keyboard. */
interface SuspendedElement {
  readonly element: HTMLElement;
  readonly hadInert: boolean;
  readonly ariaHidden: string | null;
}

/** The provider's overlay as it currently stands on the page. */
export interface OverlayElements {
  /** Containers the provider has on screen — the canvas, the frame, the dimmer. */
  readonly roots: readonly HTMLElement[];
  /** The checkout frame, once it exists. Null while the overlay is still building. */
  readonly frame: HTMLElement | null;
}

export interface OverlayKeyboardOptions {
  /**
   * The control the buyer used to start this checkout.
   *
   * Passed in rather than read off the page, because by the time the overlay
   * appears the page no longer knows. Starting a checkout puts the button into
   * its busy state, and a disabled button is blurred by the browser — several
   * hundred milliseconds before the provider's script has finished loading and
   * appended anything to observe. What this module used to capture at that
   * point was `<body>`, so dismissing the checkout dropped the buyer at the top
   * of the page with no idea where the modal they had been reading went.
   */
  readonly returnFocusTo?: HTMLElement | null;
}

export interface OverlayKeyboard {
  /**
   * Hands the keyboard to the overlay.
   *
   * Meant to be called repeatedly, on every tick of the caller's watch: the
   * canvas is appended before the frame inside it exists, so the frame is
   * focused on the first tick where there is one, and anything the app renders
   * behind the checkout in the meantime is suspended as it appears. Focus is
   * only pushed when it is not already inside the overlay.
   */
  handOver(overlay: OverlayElements): void;
  /** Gives the page its keyboard back, focus included. */
  release(): void;
}

/**
 * How long to keep waiting for the opening control to become focusable again.
 *
 * The control is re-enabled by the app's own close handling, which runs after
 * this release and settles a render later. Two seconds is far longer than that
 * takes and short enough that a checkout which left the button disabled for
 * good — an error path that keeps the busy state — does not leave an observer
 * on the page.
 */
const RETURN_FOCUS_TIMEOUT_MS = 2_000;

/** Whether `element` is on the page and able to take focus right now. */
function canTakeFocus(element: HTMLElement): boolean {
  if (!element.isConnected) return false;
  if (element.closest("[inert]")) return false;
  if ((element as { disabled?: boolean }).disabled === true) return false;
  return isRendered(element);
}

/**
 * Hands focus back to `target` as soon as it can take it, and reports how to
 * call the attempt off.
 *
 * Usually the move is immediate. It is not when the target is the control that
 * started the checkout: the app disables that button for as long as a purchase
 * is in flight, and it is re-enabled a render after whatever ended the
 * checkout. `focus()` on a disabled button is silently ignored, which is how a
 * correct-looking restore still left the buyer on `<body>`. So the move waits
 * for the attribute to come off, and gives up rather than waiting forever.
 *
 * `isStranded` is asked twice — once here and once at the moment focus is
 * actually moved — because the app may place focus itself while this waits,
 * and that decision is the newer one.
 */
export function returnFocusWhenReady(
  target: HTMLElement,
  isStranded: () => boolean,
): () => void {
  const nothingToCancel = () => {};

  if (!isStranded()) return nothingToCancel;

  if (canTakeFocus(target)) {
    target.focus();
    return nothingToCancel;
  }
  // Only a control that is merely busy is worth waiting for. One that has left
  // the page — a modal closed under the checkout — is not coming back.
  if (!target.isConnected) return nothingToCancel;

  const stop = () => {
    observer?.disconnect();
    window.clearTimeout(timeoutId);
  };

  const attempt = () => {
    if (!canTakeFocus(target)) return;
    stop();
    if (isStranded()) target.focus();
  };

  const observer =
    typeof MutationObserver === "undefined"
      ? null
      : new MutationObserver(attempt);
  // The whole body, not just the target: what makes the control focusable
  // again is as often a change on an ancestor — a modal's wrapper losing
  // `display: none`, a container losing `inert` — as one on the control
  // itself. Cheap enough to be right about, because the observer is torn down
  // at the first success and in any case two seconds from now.
  observer?.observe(document.body, {
    attributes: true,
    attributeFilter: ["disabled", "inert", "hidden", "style", "class"],
    subtree: true,
  });
  const timeoutId = window.setTimeout(stop, RETURN_FOCUS_TIMEOUT_MS);

  return stop;
}

export function createOverlayKeyboard({
  returnFocusTo,
}: OverlayKeyboardOptions = {}): OverlayKeyboard {
  let suspended: SuspendedElement[] | null = null;
  let focusBeforeOverlay: HTMLElement | null = null;
  /** The overlay focus is currently inside, so a release can tell it is leaving. */
  let overlayRoots: readonly HTMLElement[] = [];
  /** Cancels a focus return that is still waiting for its target, if any. */
  let stopWaitingForTarget: (() => void) | null = null;

  const suspendPage = (roots: readonly HTMLElement[]): void => {
    if (!suspended) {
      // The opening control if the caller named one, and only otherwise what
      // the page happens to be focusing — read before suspending, because
      // `inert` blurs whatever it covers.
      focusBeforeOverlay =
        returnFocusTo ?? (document.activeElement as HTMLElement | null);
      suspended = [];
    }

    // Re-scanned on every hand-over rather than captured once, because the
    // page keeps rendering behind the checkout: a toast portal appended while
    // the payment form is up would otherwise be the one thing on the page a
    // Tab could still reach.
    const alreadySuspended = new Set(suspended.map((entry) => entry.element));
    const newlySuspended = Array.from(document.body.children)
      .filter((child): child is HTMLElement => child instanceof HTMLElement)
      .filter((child) => !alreadySuspended.has(child))
      .filter((child) => !roots.some((root) => child.contains(root)))
      .map((element) => ({
        element,
        hadInert: element.hasAttribute("inert"),
        ariaHidden: element.getAttribute("aria-hidden"),
      }));

    for (const { element } of newlySuspended) {
      element.setAttribute("inert", "");
      element.setAttribute("aria-hidden", "true");
    }
    suspended = [...suspended, ...newlySuspended];
  };

  const focusOverlay = ({ roots, frame }: OverlayElements): void => {
    const active = document.activeElement as HTMLElement | null;
    // Focus inside a cross-origin frame is reported as the frame element
    // itself, so this is also how "the buyer is already typing" reads.
    if (active && roots.some((root) => root.contains(active))) return;
    // No frame yet: the background is already inert, so nothing reachable is
    // hidden behind the overlay. The next tick tries again.
    frame?.focus();
  };

  return {
    handOver(overlay) {
      if (overlay.roots.length === 0) return;
      // A checkout that reopens cancels any return still waiting on the old one.
      stopWaitingForTarget?.();
      suspendPage(overlay.roots);
      overlayRoots = overlay.roots;
      focusOverlay(overlay);
    },

    release() {
      if (!suspended) return;

      for (const { element, hadInert, ariaHidden } of suspended) {
        if (!hadInert) element.removeAttribute("inert");
        if (ariaHidden === null) {
          element.removeAttribute("aria-hidden");
        } else {
          element.setAttribute("aria-hidden", ariaHidden);
        }
      }
      suspended = null;

      const previous = focusBeforeOverlay;
      const roots = overlayRoots;
      focusBeforeOverlay = null;
      overlayRoots = [];

      // Focus is stranded when the page has none, or when it is still inside
      // an overlay that is on its way out — the provider fires its close
      // callback with the frame sometimes still on the page, and letting the
      // browser drop that focus would land the buyer back on `<body>`.
      //
      // Anywhere else and it is left alone: by then the app may have placed
      // focus itself, and that decision is the newer one. Re-checked at the
      // moment focus is actually moved, not just here, because the move can
      // have to wait.
      const isStranded = () => {
        const active = document.activeElement as HTMLElement | null;
        return (
          !active ||
          active === document.body ||
          roots.some((root) => root.contains(active))
        );
      };

      if (!previous) return;
      stopWaitingForTarget?.();
      stopWaitingForTarget = returnFocusWhenReady(previous, isStranded);
    },
  };
}
