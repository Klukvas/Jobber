import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

/** The dialog's accessible name, resolved the way assistive tech resolves it. */
function accessibleName(): string | null {
  const dialog = screen.getByRole("dialog");
  const labelledBy = dialog.getAttribute("aria-labelledby");
  if (!labelledBy) return null;
  return document.getElementById(labelledBy)?.textContent ?? null;
}

/**
 * Twenty-eight of the app's thirty-one dialogs opened with no accessible name:
 * every one of them drew a heading, and `aria-labelledby` was wired up by hand
 * at only the three auth modals. The contract below moves that wiring into the
 * shared components, so a dialog is named because it has a heading rather than
 * because someone remembered.
 */
describe("Dialog accessible name — the shared contract", () => {
  it("takes its name from the heading inside it", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create company</DialogTitle>
          </DialogHeader>
        </DialogContent>
      </Dialog>,
    );

    expect(accessibleName()).toBe("Create company");
  });

  it("points at the heading that is actually on screen", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>Named heading</DialogTitle>
        </DialogContent>
      </Dialog>,
    );

    const labelledBy = screen
      .getByRole("dialog")
      .getAttribute("aria-labelledby");

    expect(labelledBy).toBeTruthy();
    expect(document.getElementById(labelledBy ?? "")).toBe(
      screen.getByRole("heading", { name: "Named heading" }),
    );
  });

  // The three auth modals address their headings by a fixed id and pass it in.
  // That has to keep winning, or this change would rename them.
  it("lets an explicit labelledBy win over the heading", () => {
    render(
      <>
        <h2 id="external-heading">Sign in</h2>
        <Dialog open onOpenChange={vi.fn()} labelledBy="external-heading">
          <DialogContent>
            <DialogTitle>Ignored heading</DialogTitle>
          </DialogContent>
        </Dialog>
      </>,
    );

    expect(accessibleName()).toBe("Sign in");
  });

  it("honours a heading id the caller chose itself", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle id="fixed-title">Fixed</DialogTitle>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByRole("dialog")).toHaveAttribute(
      "aria-labelledby",
      "fixed-title",
    );
    expect(accessibleName()).toBe("Fixed");
  });

  // A reference that points at nothing reads exactly like no name at all, and
  // hides the fault. No heading means no attribute.
  it("emits no dangling reference when there is no heading", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <p>Body only</p>
        </DialogContent>
      </Dialog>,
    );

    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-labelledby");
  });

  // Several dialogs swap headings between branches — a loading state, a
  // not-connected state, the form. The name has to follow.
  it("follows the heading when the dialog swaps branches", () => {
    function Swapping({ loading }: { loading: boolean }) {
      return (
        <Dialog open onOpenChange={vi.fn()}>
          <DialogContent>
            {loading ? (
              <DialogTitle>Loading</DialogTitle>
            ) : (
              <DialogTitle>Schedule a stage</DialogTitle>
            )}
          </DialogContent>
        </Dialog>
      );
    }

    const { rerender } = render(<Swapping loading />);
    expect(accessibleName()).toBe("Loading");

    rerender(<Swapping loading={false} />);
    expect(accessibleName()).toBe("Schedule a stage");
  });

  it("never leaves two headings sharing one id", () => {
    render(
      <Dialog open onOpenChange={vi.fn()}>
        <DialogContent>
          <DialogTitle>First</DialogTitle>
          <DialogTitle>Second</DialogTitle>
        </DialogContent>
      </Dialog>,
    );

    const ids = screen
      .getAllByRole("heading")
      .map((heading) => heading.getAttribute("id"));

    expect(new Set(ids).size).toBe(ids.length);
    expect(accessibleName()).toBe("First");
  });
});
