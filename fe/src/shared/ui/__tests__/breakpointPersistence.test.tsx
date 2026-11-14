import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dialog, DialogContent } from "../Dialog";
import { MovablePanel, MovablePanelOutlet } from "../MovablePanel";
import { useMovablePanelNode } from "@/shared/hooks/useMovablePanelNode";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// matchMedia stub whose width queries can flip at runtime, like rotating a
// phone or resizing a window across a breakpoint.
let isNarrow = true;
let listeners: Array<{
  query: string;
  handler: (e: MediaQueryListEvent) => void;
}> = [];
const matchesQuery = (query: string) =>
  query.includes("max-width")
    ? isNarrow
    : query.includes("min-width")
      ? !isNarrow
      : false;
const originalMatchMedia = window.matchMedia;

function setViewport(narrow: boolean) {
  isNarrow = narrow;
  act(() => {
    listeners.forEach(({ query, handler }) =>
      handler({
        matches: matchesQuery(query),
        media: query,
      } as MediaQueryListEvent),
    );
  });
}

beforeEach(() => {
  isNarrow = true;
  listeners = [];
  window.matchMedia = ((query: string) => ({
    matches: matchesQuery(query),
    media: query,
    onchange: null,
    addEventListener: (
      _type: string,
      handler: (e: MediaQueryListEvent) => void,
    ) => listeners.push({ query, handler }),
    removeEventListener: (
      _type: string,
      handler: (e: MediaQueryListEvent) => void,
    ) => {
      listeners = listeners.filter((entry) => entry.handler !== handler);
    },
    addListener: (handler: (e: MediaQueryListEvent) => void) =>
      listeners.push({ query, handler }),
    removeListener: (handler: (e: MediaQueryListEvent) => void) => {
      listeners = listeners.filter((entry) => entry.handler !== handler);
    },
    dispatchEvent: () => false,
  })) as typeof window.matchMedia;
});

afterEach(() => {
  window.matchMedia = originalMatchMedia;
});

function Field() {
  const [value, setValue] = useState("");
  return (
    <input
      aria-label="Email"
      value={value}
      onChange={(e) => setValue(e.target.value)}
    />
  );
}

describe("Dialog across the sheet breakpoint", () => {
  // Regression: login/register forms were wiped when the viewport crossed 640px.
  it("keeps what was typed when the sheet turns into a centered dialog and back", async () => {
    render(
      <Dialog open onOpenChange={() => {}} swipeToDismiss>
        <DialogContent>
          <Field />
        </DialogContent>
      </Dialog>,
    );

    await userEvent.type(screen.getByLabelText("Email"), "qa@example.com");
    setViewport(false);
    expect(screen.getByLabelText("Email")).toHaveValue("qa@example.com");
    setViewport(true);
    expect(screen.getByLabelText("Email")).toHaveValue("qa@example.com");
  });
});

function Editor() {
  const node = useMovablePanelNode();
  const [isDesktop, setIsDesktop] = useState(false);
  return (
    <>
      <MovablePanel node={node}>
        <Field />
      </MovablePanel>
      <button onClick={() => setIsDesktop((current) => !current)}>
        switch
      </button>
      {isDesktop ? (
        <aside data-testid="column">
          <MovablePanelOutlet node={node} />
        </aside>
      ) : (
        <section data-testid="sheet">
          <MovablePanelOutlet node={node} />
        </section>
      )}
    </>
  );
}

describe("MovablePanel", () => {
  // Regression: resume-builder AI/ATS panels lost their results when moved
  // between the desktop column and the mobile sheet.
  it("moves its content between outlets without remounting it", async () => {
    render(<Editor />);
    await userEvent.type(screen.getByLabelText("Email"), "kept");
    expect(screen.getByTestId("sheet")).toContainElement(
      screen.getByLabelText("Email"),
    );

    await userEvent.click(screen.getByRole("button", { name: "switch" }));

    expect(screen.getByTestId("column")).toContainElement(
      screen.getByLabelText("Email"),
    );
    expect(screen.getByLabelText("Email")).toHaveValue("kept");
  });
});
