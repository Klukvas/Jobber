import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { Dialog, DialogTitle } from "@/shared/ui/Dialog";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

/**
 * The trap collected everything matching a focusable selector, visible or not.
 * Modals in this app routinely keep controls in the tree and hide them — a
 * save button that only appears once a field changes, a wizard step that is not
 * the current one — so the last element in that list was regularly something
 * with no box on screen. `focus()` on it does nothing, which left the keyboard
 * where it was, and Tab walked out of the modal from there.
 */
describe("dialog focus trap and hidden controls", () => {
  function renderDialog(children: React.ReactNode) {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogTitle>trap</DialogTitle>
        {children}
      </Dialog>,
    );
  }

  it("wraps backwards onto the last control that is actually on screen", () => {
    renderDialog(
      <>
        <button>first</button>
        <button>visible last</button>
        <button style={{ display: "none" }}>hidden</button>
      </>,
    );
    screen.getByText("first").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("visible last"));
  });

  it("wraps forwards from the last visible control instead of letting focus out", () => {
    renderDialog(
      <>
        <button>first</button>
        <button>visible last</button>
        <button style={{ display: "none" }}>hidden</button>
      </>,
    );
    screen.getByText("visible last").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });

  it("skips a control hidden by a wrapper rather than by its own style", () => {
    renderDialog(
      <>
        <button>first</button>
        <button>visible last</button>
        <div style={{ display: "none" }}>
          <button>inside a hidden wrapper</button>
        </div>
      </>,
    );
    screen.getByText("first").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("visible last"));
  });

  it("skips a control that is present but invisible", () => {
    renderDialog(
      <>
        <button>first</button>
        <button>visible last</button>
        <button style={{ visibility: "hidden" }}>invisible</button>
      </>,
    );
    screen.getByText("first").focus();

    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });

    expect(document.activeElement).toBe(screen.getByText("visible last"));
  });

  it("still keeps focus on the dialog when everything inside is hidden", () => {
    renderDialog(
      <div style={{ display: "none" }}>
        <button>unreachable</button>
      </div>,
    );
    const dialog = screen.getByRole("dialog");

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(dialog);
  });

  it("leaves an ordinary all-visible dialog cycling as before", () => {
    renderDialog(
      <>
        <button>first</button>
        <button>last</button>
      </>,
    );
    screen.getByText("last").focus();

    fireEvent.keyDown(document, { key: "Tab" });

    expect(document.activeElement).toBe(screen.getByText("first"));
  });
});
