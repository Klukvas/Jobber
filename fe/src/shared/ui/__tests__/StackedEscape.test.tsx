import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { Dialog, DialogTitle } from "../Dialog";
import { Sheet } from "../Sheet";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

/**
 * Every open dialog used to hold its own `document` keydown listener, and every
 * one of them fired on the same Escape. A "discard these changes?" confirmation
 * opened over a half-filled form therefore closed both — the answer and the
 * thing being asked about — and whatever had been typed was gone.
 */
describe("Escape with dialogs stacked", () => {
  afterEach(() => {
    // A leaked listener would only show up in the *next* test, so the leak
    // check below is the one that has to be explicit.
    document.body.innerHTML = "";
  });

  function Nested({
    onOuterChange,
    onInnerChange,
    innerOpen = true,
  }: {
    onOuterChange: (open: boolean) => void;
    onInnerChange: (open: boolean) => void;
    innerOpen?: boolean;
  }) {
    return (
      <Dialog open onOpenChange={onOuterChange}>
        <DialogTitle>outer</DialogTitle>
        <Dialog open={innerOpen} onOpenChange={onInnerChange}>
          <DialogTitle>inner</DialogTitle>
        </Dialog>
      </Dialog>
    );
  }

  it("closes only the dialog on top", () => {
    const onOuterChange = vi.fn();
    const onInnerChange = vi.fn();
    render(
      <Nested onOuterChange={onOuterChange} onInnerChange={onInnerChange} />,
    );

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onInnerChange).toHaveBeenCalledWith(false);
    expect(onOuterChange).not.toHaveBeenCalled();
  });

  it("closes the one underneath on the next Escape", () => {
    const onOuterChange = vi.fn();
    const onInnerChange = vi.fn();
    const { rerender } = render(
      <Nested onOuterChange={onOuterChange} onInnerChange={onInnerChange} />,
    );
    fireEvent.keyDown(document, { key: "Escape" });

    // The app answers the inner dialog's request and takes it off the page.
    rerender(
      <Nested
        onOuterChange={onOuterChange}
        onInnerChange={onInnerChange}
        innerOpen={false}
      />,
    );
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOuterChange).toHaveBeenCalledWith(false);
    expect(onOuterChange).toHaveBeenCalledTimes(1);
  });

  it("leaves no listener behind when the stack unmounts", () => {
    const onOuterChange = vi.fn();
    const onInnerChange = vi.fn();
    const { unmount } = render(
      <Nested onOuterChange={onOuterChange} onInnerChange={onInnerChange} />,
    );

    unmount();
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onInnerChange).not.toHaveBeenCalled();
    expect(onOuterChange).not.toHaveBeenCalled();
  });

  it("hands Escape back to a lone dialog once the stack has drained", () => {
    const onOpenChange = vi.fn();
    const stack = render(
      <Nested onOuterChange={vi.fn()} onInnerChange={vi.fn()} />,
    );
    stack.unmount();

    render(
      <Dialog open onOpenChange={onOpenChange}>
        <DialogTitle>alone</DialogTitle>
      </Dialog>,
    );
    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  // A dialog opened over a mobile sheet is the same stack, and the sheet
  // closing under it would strand the dialog on a page that had moved on.
  it("does not close the sheet underneath a dialog", () => {
    const onSheetChange = vi.fn();
    const onDialogChange = vi.fn();
    render(
      <Sheet open onOpenChange={onSheetChange} title="sheet">
        <Dialog open onOpenChange={onDialogChange}>
          <DialogTitle>over the sheet</DialogTitle>
        </Dialog>
      </Sheet>,
    );

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onDialogChange).toHaveBeenCalledWith(false);
    expect(onSheetChange).not.toHaveBeenCalled();
  });

  // An overlay outside this app's DOM cannot take its place in the stack; the
  // dialog behind it opts out entirely instead.
  it("lets the dialog underneath answer when the top one has opted out", () => {
    const onOuterChange = vi.fn();
    const onInnerChange = vi.fn();
    render(
      <Dialog open onOpenChange={onOuterChange}>
        <DialogTitle>outer</DialogTitle>
        <Dialog open hasExternalOverlay onOpenChange={onInnerChange}>
          <DialogTitle>inner</DialogTitle>
        </Dialog>
      </Dialog>,
    );

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onInnerChange).not.toHaveBeenCalled();
    expect(onOuterChange).toHaveBeenCalledWith(false);
  });
});

/**
 * Two sheets on the page used to render two `id="sheet-heading"` attributes,
 * so both `aria-labelledby` references resolved to the same node — whichever
 * the document happened to hold first. One sheet was announced with the other's
 * title.
 */
describe("Sheet accessible names", () => {
  it("gives each sheet a heading of its own to point at", () => {
    render(
      <>
        <Sheet open onOpenChange={vi.fn()} title="first sheet">
          <p>one</p>
        </Sheet>
        <Sheet open onOpenChange={vi.fn()} title="second sheet">
          <p>two</p>
        </Sheet>
      </>,
    );

    const [first, second] = screen.getAllByRole("dialog");
    const firstId = first.getAttribute("aria-labelledby");
    const secondId = second.getAttribute("aria-labelledby");

    expect(firstId).toBeTruthy();
    expect(secondId).toBeTruthy();
    expect(firstId).not.toBe(secondId);
    expect(document.getElementById(firstId!)).toHaveTextContent("first sheet");
    expect(document.getElementById(secondId!)).toHaveTextContent(
      "second sheet",
    );
  });

  it("emits no dangling reference when the sheet has no title", () => {
    render(
      <Sheet open onOpenChange={vi.fn()}>
        <p>untitled</p>
      </Sheet>,
    );

    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-labelledby");
  });
});
