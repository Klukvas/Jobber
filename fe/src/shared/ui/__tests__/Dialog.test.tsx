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
  // The billing provider's payment popup is appended to <body>, outside this
  // dialog. Trapping Tab would lock the keyboard out of the payment form, and
  // Escape would close the page behind a payment that is already in flight.
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
});
