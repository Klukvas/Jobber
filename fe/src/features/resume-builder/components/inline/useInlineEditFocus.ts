import * as React from "react";

/**
 * Input types with a defined text selection.
 *
 * `setSelectionRange` throws `InvalidStateError` on the others — `date` and
 * `email` among them, both of which these fields render — so the caret is only
 * placed where the platform allows one.
 */
const SELECTABLE_INPUT_TYPES = new Set([
  "text",
  "search",
  "url",
  "tel",
  "password",
]);

type InlineEditor = HTMLInputElement | HTMLTextAreaElement;

function hasTextSelection(element: InlineEditor): boolean {
  return (
    element instanceof HTMLTextAreaElement ||
    SELECTABLE_INPUT_TYPES.has(element.type)
  );
}

/**
 * Focuses an inline editor and leaves the caret at the end of its text.
 *
 * The caret is placed explicitly because the click that opened the field never
 * reaches it: the display element calls `preventDefault` on `mousedown` to
 * keep the browser from starting a text selection on it, which also discards
 * the caret position that click would have carried. Without a decision here
 * the field opens with an undefined selection, and where the first keystroke
 * lands is up to the browser.
 */
function focusEditor(element: InlineEditor | null): void {
  if (!element) return;

  element.focus();
  if (document.activeElement !== element) return;
  if (!hasTextSelection(element)) return;

  const end = element.value.length;
  element.setSelectionRange(end, end);
}

/**
 * Puts the keyboard in an inline field the moment it opens.
 *
 * This used to focus from a `requestAnimationFrame` callback that closed over
 * the element it had been given, and both halves of that were wrong.
 *
 * The frame is a frame the field does not control. Committing the field the
 * visitor is leaving writes to the resume store, which re-renders the whole
 * preview — so by the time the callback ran, the node it was holding could
 * already have been replaced by an equal one. `focus()` on a detached element
 * does nothing at all and silently drops focus on `<body>`: clicking straight
 * from one field to another left the caret nowhere, or still in the field just
 * left, and what was typed next went into the wrong field or into no field.
 *
 * So focus is taken synchronously in a layout effect, before the browser
 * paints the frame the input first appears in, and the one retry that remains
 * — for a container still being laid out, which can refuse focus for a frame —
 * re-reads the ref instead of trusting a captured node, and only acts if the
 * first attempt did not take.
 */
export function useInlineEditFocus(
  editorRef: React.RefObject<InlineEditor | null>,
  isEditing: boolean,
): void {
  React.useLayoutEffect(() => {
    if (!isEditing) return;

    focusEditor(editorRef.current);

    const frame = requestAnimationFrame(() => {
      const element = editorRef.current;
      if (element && document.activeElement !== element) focusEditor(element);
    });
    return () => cancelAnimationFrame(frame);
  }, [isEditing, editorRef]);
}
