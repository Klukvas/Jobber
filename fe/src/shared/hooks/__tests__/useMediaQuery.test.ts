import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useMediaQuery } from "../useMediaQuery";

describe("useMediaQuery", () => {
  let listeners: Array<(e: MediaQueryListEvent) => void>;
  let matchesValue: boolean;

  beforeEach(() => {
    listeners = [];
    matchesValue = false;

    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: matchesValue,
        media: query,
        addEventListener: (_event: string, handler: (e: MediaQueryListEvent) => void) => {
          listeners.push(handler);
        },
        removeEventListener: (_event: string, handler: (e: MediaQueryListEvent) => void) => {
          listeners = listeners.filter((l) => l !== handler);
        },
      })),
    });
  });

  it("returns false when media query does not match", () => {
    matchesValue = false;
    const { result } = renderHook(() =>
      useMediaQuery("(min-width: 1024px)"),
    );
    expect(result.current).toBe(false);
  });

  it("returns true when media query matches", () => {
    matchesValue = true;
    const { result } = renderHook(() =>
      useMediaQuery("(min-width: 1024px)"),
    );
    expect(result.current).toBe(true);
  });

  it("updates when the media query change event fires", () => {
    matchesValue = false;
    const { result } = renderHook(() =>
      useMediaQuery("(min-width: 1024px)"),
    );
    expect(result.current).toBe(false);

    act(() => {
      for (const listener of listeners) {
        listener({ matches: true } as MediaQueryListEvent);
      }
    });

    expect(result.current).toBe(true);
  });

  it("cleans up event listener on unmount", () => {
    const { unmount } = renderHook(() =>
      useMediaQuery("(min-width: 768px)"),
    );
    expect(listeners.length).toBe(1);

    unmount();
    expect(listeners.length).toBe(0);
  });

  it("calls matchMedia with the correct query string", () => {
    renderHook(() => useMediaQuery("(max-width: 640px)"));
    expect(window.matchMedia).toHaveBeenCalledWith("(max-width: 640px)");
  });

  it("re-subscribes when query changes", () => {
    const { rerender } = renderHook(
      ({ query }) => useMediaQuery(query),
      { initialProps: { query: "(min-width: 768px)" } },
    );
    expect(listeners.length).toBe(1);

    rerender({ query: "(min-width: 1024px)" });
    // Old listener removed, new one added
    expect(window.matchMedia).toHaveBeenCalledWith("(min-width: 1024px)");
  });
});
