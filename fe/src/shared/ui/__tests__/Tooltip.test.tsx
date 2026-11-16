import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { Tooltip } from "../Tooltip";

describe("Tooltip", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows after the delay", () => {
    render(
      <Tooltip content="Undo" delayMs={300}>
        <button type="button">u</button>
      </Tooltip>,
    );

    fireEvent.mouseEnter(screen.getByRole("button").parentElement as HTMLElement);
    expect(screen.queryByRole("tooltip")).toBeNull();

    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(screen.getByRole("tooltip")).toHaveTextContent("Undo");
  });

  // Focus and hover both start the show timer. The second must replace the
  // first, or the first outlives unmount and fires after the DOM is gone.
  it("leaves no pending timer after focus and hover, then unmount", () => {
    const { unmount } = render(
      <Tooltip content="Undo" delayMs={300}>
        <button type="button">u</button>
      </Tooltip>,
    );
    const wrapper = screen.getByRole("button").parentElement as HTMLElement;

    fireEvent.focus(wrapper);
    fireEvent.mouseEnter(wrapper);
    unmount();

    expect(vi.getTimerCount()).toBe(0);
  });
});
