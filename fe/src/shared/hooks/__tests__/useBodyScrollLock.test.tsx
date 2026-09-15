import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render } from "@testing-library/react";

import { useBodyScrollLock } from "../useBodyScrollLock";

function Overlay({ open }: { readonly open: boolean }) {
  useBodyScrollLock(open);
  return null;
}

/**
 * Every overlay used to keep its own copy of `document.body.style.overflow`,
 * taken when it opened and written back when it closed. That is only correct
 * while overlays close in the order they opened: a drawer opened first and
 * closed second put back the value it had captured before the dialog over it
 * locked, and the page scrolled underneath a modal that was still up.
 */
describe("useBodyScrollLock", () => {
  beforeEach(() => {
    document.body.style.overflow = "";
  });

  afterEach(() => {
    document.body.style.overflow = "";
  });

  it("freezes the page while an overlay is up", () => {
    render(<Overlay open />);

    expect(document.body.style.overflow).toBe("hidden");
  });

  it("leaves the page alone while nothing is up", () => {
    render(<Overlay open={false} />);

    expect(document.body.style.overflow).toBe("");
  });

  it("gives the page back when the overlay closes", () => {
    const { rerender } = render(<Overlay open />);

    rerender(<Overlay open={false} />);

    expect(document.body.style.overflow).toBe("");
  });

  it("gives the page back when the overlay unmounts", () => {
    const { unmount } = render(<Overlay open />);

    unmount();

    expect(document.body.style.overflow).toBe("");
  });

  it("restores whatever the page had before the first lock", () => {
    document.body.style.overflow = "clip";
    const { unmount } = render(<Overlay open />);

    unmount();

    expect(document.body.style.overflow).toBe("clip");
  });

  it("keeps the page frozen while a second overlay is still up", () => {
    const drawer = render(<Overlay open />);
    const dialog = render(<Overlay open />);

    drawer.unmount();

    // The drawer opened first and closed first; the dialog over it is still on
    // screen, so nothing behind it may start scrolling again.
    expect(document.body.style.overflow).toBe("hidden");

    dialog.unmount();
    expect(document.body.style.overflow).toBe("");
  });

  it("keeps the page frozen when overlays close out of order", () => {
    const drawer = render(<Overlay open />);
    const dialog = render(<Overlay open />);

    dialog.unmount();
    expect(document.body.style.overflow).toBe("hidden");

    drawer.unmount();
    expect(document.body.style.overflow).toBe("");
  });
});
