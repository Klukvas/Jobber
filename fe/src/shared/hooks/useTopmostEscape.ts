import * as React from "react";

/**
 * One open overlay's claim on the Escape key.
 *
 * The element is read at keypress time rather than captured, because a dialog
 * registers before its container is attached and can re-render into a
 * different node afterwards.
 */
interface EscapeClaim {
  readonly containerRef: React.RefObject<HTMLElement | null>;
}

/**
 * Every overlay currently listening for Escape.
 *
 * Module scope, because the question "am I the one that should close?" cannot
 * be answered from inside a single component: each open dialog attached its own
 * `document` keydown listener, every one of them fired on the same keypress,
 * and one Escape closed the whole stack — a confirmation opened over a form
 * took the form down with it, losing what had been typed.
 *
 * Replaced rather than mutated, so a claim removed while the array is being
 * walked cannot change the walk underneath it.
 */
let claims: readonly EscapeClaim[] = [];

/** Whether `a` comes after `b` in document order — a descendant counts. */
function followsInDocument(a: HTMLElement, b: HTMLElement): boolean {
  const relation = b.compareDocumentPosition(a);
  return (relation & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
}

/**
 * Whether `claim` is the overlay on top.
 *
 * Decided from the DOM rather than from the order the claims were registered.
 * Effects run innermost-first, so a page that mounts both dialogs in one commit
 * registers the *inner* one first — registration order would have made the
 * outer dialog "topmost" and closed the wrong one. Document order has no such
 * ambiguity: a nested dialog is a descendant of the one it sits over, and a
 * descendant follows its ancestor; two unrelated overlays are ordered by which
 * was rendered later, which is the one the user is looking at.
 *
 * A claim whose container is not on the page yet is treated as topmost only if
 * it is alone — nothing about it can be compared, and closing every dialog is
 * the failure this exists to prevent.
 */
function isTopmost(claim: EscapeClaim): boolean {
  const element = claim.containerRef.current;
  const others = claims.filter((other) => other !== claim);
  if (others.length === 0) return true;
  if (!element?.isConnected) return false;

  return others.every((other) => {
    const otherElement = other.containerRef.current;
    if (!otherElement?.isConnected) return true;
    return followsInDocument(element, otherElement);
  });
}

/**
 * Closes the topmost overlay on Escape, and only the topmost.
 *
 * `active` is what a dialog uses to opt out entirely — while it is false the
 * overlay neither listens nor takes its place in the stack, so Escape falls
 * through to whatever is underneath.
 */
export function useTopmostEscape(
  active: boolean,
  containerRef: React.RefObject<HTMLElement | null>,
  onEscape: () => void,
): void {
  // Held in a ref so the claim is registered exactly once per open overlay. A
  // parent re-render gives `onEscape` a new identity, and re-running the effect
  // for that would drop this claim and add it back — reordering the stack for a
  // reason that has nothing to do with what is on screen.
  const onEscapeRef = React.useRef(onEscape);
  React.useLayoutEffect(() => {
    onEscapeRef.current = onEscape;
  });

  React.useEffect(() => {
    if (!active) return;

    const claim: EscapeClaim = { containerRef };
    claims = [...claims, claim];

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      if (!isTopmost(claim)) return;
      onEscapeRef.current();
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      claims = claims.filter((entry) => entry !== claim);
    };
  }, [active, containerRef]);
}
