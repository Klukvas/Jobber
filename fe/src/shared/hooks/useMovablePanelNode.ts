import { useState } from "react";

/**
 * Content that has to appear in different places of the page — a side column
 * on desktop, a sheet on phones — without remounting when it moves.
 *
 * Rendering the same element in two places means two React mount points, so
 * crossing the breakpoint unmounts one copy and mounts a fresh one, discarding
 * everything it held (typed text, an unapplied AI result). Here the content is
 * rendered once, into a detached DOM node, at a fixed spot in the React tree;
 * `MovablePanelOutlet` only moves that node into whichever host is on screen.
 */
export function useMovablePanelNode(): HTMLElement {
  const [node] = useState(() => {
    const element = document.createElement("div");
    // The node is an implementation detail; it must not add a layout box.
    element.style.display = "contents";
    return element;
  });
  return node;
}
