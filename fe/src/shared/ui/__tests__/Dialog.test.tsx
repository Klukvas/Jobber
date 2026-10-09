import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "../Dialog";
import { Sheet } from "../Sheet";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

// ---------- Dialog ----------
describe("Dialog", () => {
  it("renders children when open=true", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <div>Dialog content</div>
      </Dialog>,
    );
    expect(screen.getByText("Dialog content")).toBeInTheDocument();
  });

  it("returns null when open=false", () => {
    const { container } = render(
      <Dialog open={false} onOpenChange={vi.fn()}>
        <div>Dialog content</div>
      </Dialog>,
    );
    expect(container.innerHTML).toBe("");
  });

  it("has role=dialog and aria-modal when open", () => {
    render(
      <Dialog open={true} onOpenChange={vi.fn()}>
        <div>content</div>
      </Dialog>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
  });

  it("calls onOpenChange(false) when backdrop is clicked", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <div>content</div>
      </Dialog>,
    );
    // Click the outer fixed wrapper (backdrop area)
    const backdrop = screen.getByRole("dialog").parentElement!;
    fireEvent.click(backdrop);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("calls onOpenChange(false) when Escape is pressed", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <div>content</div>
      </Dialog>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not propagate click from dialog content to backdrop", () => {
    const onOpenChange = vi.fn();
    render(
      <Dialog open={true} onOpenChange={onOpenChange}>
        <div>inner</div>
      </Dialog>,
    );
    fireEvent.click(screen.getByText("inner"));
    expect(onOpenChange).not.toHaveBeenCalled();
  });
});

// ---------- Dialog with an overlay it does not own ----------
describe("Dialog while an external overlay is on screen", () => {
  // An overlay (or an uninterruptible flow such as a checkout redirect) must
  // own the keyboard: trapping Tab would lock focus out of it, and Escape
  // would close the dialog underneath it.
  function renderDialog(hasExternalOverlay: boolean) {
    const onOpenChange = vi.fn();
    const view = render(
      <Dialog
        open
        onOpenChange={onOpenChange}
        hasExternalOverlay={hasExternalOverlay}
      >
        <button>first</button>
        <button>last</button>
      </Dialog>,
    );
    return { onOpenChange, ...view };
  }

  it("wraps Tab back to the first control while it owns the keyboard", () => {
    renderDialog(false);
    const last = screen.getByText("last");
    last.focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });

  it("lets Tab leave, so focus can reach the payment form", () => {
    renderDialog(true);
    const last = screen.getByText("last");
    last.focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(last);
  });

  it("ignores Escape, so it cannot close behind a payment in flight", () => {
    const { onOpenChange } = renderDialog(true);

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("takes the keyboard back when the overlay closes", () => {
    const { onOpenChange, rerender } = renderDialog(true);
    const last = screen.getByText("last");
    last.focus();

    rerender(
      <Dialog open onOpenChange={onOpenChange} hasExternalOverlay={false}>
        <button>first</button>
        <button>last</button>
      </Dialog>,
    );

    // Handing the keyboard over and back must not re-run the dialog's
    // open-time focus, or it would snatch focus mid-payment.
    expect(document.activeElement).toBe(last);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

/**
 * The auth modal opened with `document.activeElement` still on `<body>`, and a
 * real Tab from there walked into the navbar behind it. Two separate faults:
 * focus was moved in from a `requestAnimationFrame` callback the dialog does
 * not control, and the trap only acted when focus was already sitting on the
 * first or last control inside — so from anywhere else it did nothing at all.
 */
describe("Dialog keyboard containment", () => {
  function renderWithOutsideControl(open = true) {
    const onOpenChange = vi.fn();
    const view = render(
      <>
        <button>navbar</button>
        <Dialog open={open} onOpenChange={onOpenChange}>
          <button>first</button>
          <button>last</button>
        </Dialog>
      </>,
    );
    return { onOpenChange, ...view };
  }

  it("moves focus inside on open instead of leaving it on body", () => {
    expect(document.activeElement).toBe(document.body);

    renderWithOutsideControl();

    expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(
      true,
    );
  });

  it("pulls Tab back in when focus starts outside the dialog", () => {
    renderWithOutsideControl();
    screen.getByText("navbar").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });

  it("pulls Shift+Tab back to the last control when focus starts outside", () => {
    renderWithOutsideControl();
    screen.getByText("navbar").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("last"));
  });

  // The container holds focus on open and is not itself a tab stop, so both
  // directions have to be steered off it explicitly — Shift+Tab from there used
  // to leave the dialog entirely.
  it("steers Tab off the dialog container into the first control", () => {
    renderWithOutsideControl();
    screen.getByRole("dialog").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });

  it("steers Shift+Tab off the dialog container onto the last control", () => {
    renderWithOutsideControl();
    screen.getByRole("dialog").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("last"));
  });

  it("wraps Shift+Tab from the first control round to the last", () => {
    renderWithOutsideControl();
    screen.getByText("first").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("last"));
  });

  it("keeps focus on a dialog that has nothing to tab between", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <p>nothing focusable here</p>
      </Dialog>,
    );
    const dialog = screen.getByRole("dialog");

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(dialog);
  });

  it("skips a disabled control when wrapping", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <button>first</button>
        <button disabled>disabled</button>
      </Dialog>,
    );
    screen.getByText("first").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });

  it("hands focus back to whatever opened it", () => {
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <>
        <button>opener</button>
        <Dialog open={false} onOpenChange={onOpenChange}>
          <button>inside</button>
        </Dialog>
      </>,
    );
    const opener = screen.getByText("opener");
    opener.focus();

    rerender(
      <>
        <button>opener</button>
        <Dialog open onOpenChange={onOpenChange}>
          <button>inside</button>
        </Dialog>
      </>,
    );
    expect(document.activeElement).not.toBe(opener);

    rerender(
      <>
        <button>opener</button>
        <Dialog open={false} onOpenChange={onOpenChange}>
          <button>inside</button>
        </Dialog>
      </>,
    );

    expect(document.activeElement).toBe(opener);
  });

  // A route-driven modal can leave its trigger unmounted; focusing a detached
  // node silently drops focus to <body> instead of throwing, so the restore has
  // to check before it reaches for it.
  it("does not reach for an opener that has since been unmounted", () => {
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <>
        <button>trigger</button>
        <Dialog open onOpenChange={onOpenChange}>
          <button>inside</button>
        </Dialog>
      </>,
    );

    expect(() =>
      rerender(
        <Dialog open={false} onOpenChange={onOpenChange}>
          <button>inside</button>
        </Dialog>,
      ),
    ).not.toThrow();
  });
});

// ---------- DialogContent ----------
describe("DialogContent", () => {
  it("renders children", () => {
    render(<DialogContent>Hello</DialogContent>);
    expect(screen.getByText("Hello")).toBeInTheDocument();
  });

  it("renders close button when onClose is provided", () => {
    const onClose = vi.fn();
    render(<DialogContent onClose={onClose}>Body</DialogContent>);
    const btn = screen.getByLabelText("common.close");
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not render close button when onClose is omitted", () => {
    render(<DialogContent>Body</DialogContent>);
    expect(screen.queryByLabelText("common.close")).not.toBeInTheDocument();
  });

  /**
   * The X glyph is 16×16. On its own that was the entire tap target for the
   * one control every modal has to offer, well under the 44px a thumb needs.
   * The button is now a 44×44 box on phones, shrinking to a compact 32×32 from
   * `sm` up — asserted on the classes because jsdom has no layout to measure.
   */
  it("gives the close button a 44px tap target on phones", () => {
    render(<DialogContent onClose={vi.fn()}>Body</DialogContent>);
    const btn = screen.getByLabelText("common.close");

    expect(btn.className).toContain("h-11");
    expect(btn.className).toContain("w-11");
    // Centres the glyph inside the larger box rather than pinning it corner-wise.
    expect(btn.className).toContain("items-center");
    expect(btn.className).toContain("justify-center");
  });

  it("keeps the compact size on desktop", () => {
    render(<DialogContent onClose={vi.fn()}>Body</DialogContent>);
    const btn = screen.getByLabelText("common.close");

    expect(btn.className).toContain("sm:h-8");
    expect(btn.className).toContain("sm:w-8");
  });

  // The offsets exist to cancel the box growth: 2 + 22 and 8 + 16 both put the
  // glyph's centre 24px in from the corner, exactly where it was before.
  it("offsets the box so the glyph does not move", () => {
    render(<DialogContent onClose={vi.fn()}>Body</DialogContent>);
    const btn = screen.getByLabelText("common.close");

    expect(btn.className).toContain("right-0.5");
    expect(btn.className).toContain("sm:right-2");
  });

  it("hides the decorative glyph from assistive tech, leaving only the label", () => {
    render(<DialogContent onClose={vi.fn()}>Body</DialogContent>);
    const btn = screen.getByLabelText("common.close");

    expect(btn.querySelector("svg")).toHaveAttribute("aria-hidden");
  });
});

// ---------- DialogHeader / DialogTitle / DialogDescription / DialogFooter ----------
describe("Dialog sub-components", () => {
  it("renders DialogHeader", () => {
    render(<DialogHeader data-testid="dh">header</DialogHeader>);
    expect(screen.getByTestId("dh")).toHaveTextContent("header");
  });

  it("renders DialogTitle as h2", () => {
    render(<DialogTitle>My Title</DialogTitle>);
    const el = screen.getByText("My Title");
    expect(el.tagName).toBe("H2");
  });

  it("renders DialogDescription as p", () => {
    render(<DialogDescription>desc</DialogDescription>);
    const el = screen.getByText("desc");
    expect(el.tagName).toBe("P");
  });

  it("renders DialogFooter", () => {
    render(<DialogFooter data-testid="df">foot</DialogFooter>);
    expect(screen.getByTestId("df")).toHaveTextContent("foot");
  });
});

// ---------- Sheet ----------
describe("Sheet", () => {
  it("renders children when open=true", () => {
    render(
      <Sheet open={true} onOpenChange={vi.fn()}>
        <div>Sheet content</div>
      </Sheet>,
    );
    expect(screen.getByText("Sheet content")).toBeInTheDocument();
  });

  it("returns null when open=false", () => {
    const { container } = render(
      <Sheet open={false} onOpenChange={vi.fn()}>
        <div>Sheet content</div>
      </Sheet>,
    );
    expect(container.innerHTML).toBe("");
  });

  it("has role=dialog and aria-modal when open", () => {
    render(
      <Sheet open={true} onOpenChange={vi.fn()}>
        <div>content</div>
      </Sheet>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
  });

  it("renders title", () => {
    render(
      <Sheet open={true} onOpenChange={vi.fn()} title="My Sheet">
        <div>content</div>
      </Sheet>,
    );
    expect(screen.getByText("My Sheet")).toBeInTheDocument();
  });

  it("calls onOpenChange(false) when close button is clicked", () => {
    const onOpenChange = vi.fn();
    render(
      <Sheet open={true} onOpenChange={onOpenChange} title="T">
        <div>content</div>
      </Sheet>,
    );
    fireEvent.click(screen.getByLabelText("common.close"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("calls onOpenChange(false) when Escape is pressed", () => {
    const onOpenChange = vi.fn();
    render(
      <Sheet open={true} onOpenChange={onOpenChange}>
        <div>content</div>
      </Sheet>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("calls onOpenChange(false) when backdrop is clicked", () => {
    const onOpenChange = vi.fn();
    render(
      <Sheet open={true} onOpenChange={onOpenChange}>
        <div>content</div>
      </Sheet>,
    );
    // Click the outer fixed container (backdrop)
    const outer = screen.getByRole("dialog").parentElement!;
    fireEvent.click(outer);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  // The sheet shares Dialog's containment, so it has to keep the same promises.
  it("moves focus inside on open instead of leaving it on body", () => {
    render(
      <Sheet open onOpenChange={vi.fn()} title="T">
        <button>inside</button>
      </Sheet>,
    );

    expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(
      true,
    );
  });

  it("pulls Tab back in when focus starts outside the sheet", () => {
    render(
      <>
        <button>navbar</button>
        <Sheet open onOpenChange={vi.fn()}>
          <button>inside</button>
        </Sheet>
      </>,
    );
    screen.getByText("navbar").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(
      true,
    );
  });

  // The close button was a 24x24 target on a surface that only ever renders on
  // a touch layout.
  it("gives the close button a 44px tap target", () => {
    render(
      <Sheet open onOpenChange={vi.fn()} title="T">
        <div>content</div>
      </Sheet>,
    );
    const close = screen.getByLabelText("common.close");

    expect(close.className).toContain("h-11");
    expect(close.className).toContain("w-11");
  });
});
