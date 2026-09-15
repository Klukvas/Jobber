/**
 * jsdom hands `focus()` to anything with a tabindex, on screen or not — its
 * focusable-area check looks at `disabled` and the `hidden` attribute and never
 * at the computed `display` or `visibility`. Browsers refuse, and that
 * difference hides a whole class of bug: an overlay that asks for focus while
 * it is still `visibility: hidden` gets it in a test and does not get it on a
 * phone.
 *
 * Installs the browser's rule for the length of a test and returns the undo —
 * call it in `afterEach`, the prototype is shared with every other test.
 *
 * `defineProperty` rather than assignment, and the descriptor is put back as it
 * was found: `userEvent.setup()` replaces `focus` with a getter-only accessor
 * of its own, and a plain assignment over that one throws.
 */
export function installBrowserFocusRules(): () => void {
  const original = Object.getOwnPropertyDescriptor(
    HTMLElement.prototype,
    "focus",
  );
  if (!original) {
    throw new Error("No HTMLElement.prototype.focus to wrap — not a DOM test");
  }

  // Read through whatever is installed now, so a wrapper already in place
  // (user-event's, which is what dispatches `focusin`) still runs underneath.
  const currentFocus = HTMLElement.prototype.focus;

  Object.defineProperty(HTMLElement.prototype, "focus", {
    configurable: true,
    writable: true,
    value: function focusOnlyWhenOnScreen(
      this: HTMLElement,
      options?: FocusOptions,
    ): void {
      if (!isOnScreen(this)) return;
      currentFocus.call(this, options);
    },
  });

  return () => {
    Object.defineProperty(HTMLElement.prototype, "focus", original);
  };
}

/**
 * Whether the element has a box to put a focus ring on. The walk goes up
 * because `display: none` or `visibility: hidden` on an ancestor takes the
 * subtree off the screen without changing anything computed on the element.
 */
function isOnScreen(element: HTMLElement): boolean {
  for (
    let node: HTMLElement | null = element;
    node;
    node = node.parentElement
  ) {
    const { display, visibility } = window.getComputedStyle(node);
    if (display === "none" || visibility === "hidden") return false;
  }
  return true;
}
