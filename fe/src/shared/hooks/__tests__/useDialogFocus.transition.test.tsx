import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { act, render } from "@testing-library/react";
import * as React from "react";

import { useDialogFocus } from "../useDialogFocus";
import { installBrowserFocusRules } from "@/test/browserFocusability";
import {
  installManualAnimationFrames,
  type ManualAnimationFrames,
} from "@/test/animationFrames";

const PANEL_ID = "transitioning-panel";

function Panel({ open }: { open: boolean }) {
  const panelRef = React.useRef<HTMLDivElement>(null);
  useDialogFocus(panelRef, { open });

  return (
    <div id={PANEL_ID} ref={panelRef} tabIndex={-1}>
      <button>inside</button>
    </div>
  );
}

/**
 * An overlay does not become focusable the instant it is told to open. The
 * navigation drawer is `invisible` while closed and transitions every property
 * it has, `visibility` included, so at the transition's first sample the panel
 * still computes to `visibility: hidden` — and a hidden element refuses
 * `focus()`, as does everything inside it.
 *
 * The open-focus routine used to get two attempts, both of which could land on
 * that sample: the layout effect, and one frame later. When they did, no focus
 * moved at all — no `focusin`, the keyboard left on the control that opened the
 * panel, underneath the panel now covering it.
 */
describe("a dialog that is still transitioning onto the screen", () => {
  let frames: ManualAnimationFrames;
  let restoreFocusRules: () => void;
  let opener: HTMLButtonElement;

  beforeEach(() => {
    restoreFocusRules = installBrowserFocusRules();
    frames = installManualAnimationFrames();
    opener = document.createElement("button");
    document.body.appendChild(opener);
  });

  afterEach(() => {
    frames.restore();
    restoreFocusRules();
    opener.remove();
  });

  /** Opens the panel while it is still hidden, the way a real one opens. */
  function openWhileHidden() {
    const view = render(<Panel open={false} />);
    const panel = document.getElementById(PANEL_ID) as HTMLElement;
    panel.style.visibility = "hidden";
    opener.focus();

    act(() => view.rerender(<Panel open />));

    return { panel, view };
  }

  it("moves focus in as soon as the panel can take it", () => {
    const { panel } = openWhileHidden();
    // Nothing has moved yet, and nothing could have: the panel has no box.
    expect(document.activeElement).toBe(opener);

    act(() => frames.runFrame());
    // The transition passes its first sample and the panel is on screen.
    panel.style.visibility = "visible";
    act(() => frames.runFrame());

    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("keeps waiting across several frames of transition", () => {
    const { panel } = openWhileHidden();

    act(() => frames.runFrame());
    act(() => frames.runFrame());
    act(() => frames.runFrame());
    panel.style.visibility = "visible";
    act(() => frames.runFrame());

    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("takes nothing from elsewhere while the panel has no box", () => {
    openWhileHidden();
    const elsewhere = document.createElement("button");
    document.body.appendChild(elsewhere);
    elsewhere.focus();

    act(() => frames.runFrame());
    act(() => frames.runFrame());

    expect(document.activeElement).toBe(elsewhere);
    elsewhere.remove();
  });

  it("gives up on a panel that never arrives instead of retrying forever", () => {
    openWhileHidden();

    for (let attempt = 0; attempt < 60; attempt += 1) {
      act(() => frames.runFrame());
    }

    expect(frames.pendingFrames()).toBe(0);
    expect(document.activeElement).toBe(opener);
  });

  it("stops chasing the panel when the dialog closes first", () => {
    const { panel, view } = openWhileHidden();

    act(() => view.rerender(<Panel open={false} />));
    panel.style.visibility = "visible";
    act(() => frames.runFrame());
    act(() => frames.runFrame());

    expect(document.activeElement).toBe(opener);
  });
});
