/**
 * Whether an element and every ancestor is actually rendered.
 *
 * Three places need the same answer and used to give three different ones: the
 * focus trap deciding what Tab can reach, the checkout overlay deciding where
 * to hand the keyboard back, and the checkout watch deciding whether the
 * provider's checkout is still on screen. The cheap versions — reading
 * `element.style`, or `offsetParent` — are each wrong in a way that matters:
 * `style` sees nothing a stylesheet class did, and `offsetParent` is null for
 * `position: fixed` in browsers and for everything in jsdom.
 *
 * The walk goes up because `display: none` on an ancestor takes the whole
 * subtree off the screen without changing anything computed on the element
 * itself.
 */
export function isRendered(element: HTMLElement): boolean {
  for (
    let node: HTMLElement | null = element;
    node;
    node = node.parentElement
  ) {
    if (node.hidden) return false;
    const style = window.getComputedStyle?.(node);
    if (!style) {
      // No computed styles to read — a document with no view. The inline
      // declarations are all there is, and they are better than nothing.
      if (node.style.display === "none") return false;
      if (node.style.visibility === "hidden") return false;
      continue;
    }
    if (style.display === "none") return false;
    if (style.visibility === "hidden" || style.visibility === "collapse") {
      return false;
    }
  }
  return true;
}
