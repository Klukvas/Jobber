import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  useOnboarding,
  useOnboardingHighlight,
  setOnboardingHighlight,
} from "../useOnboarding";

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isAuthenticated: true }),
}));

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
});

describe("useOnboarding", () => {
  it("shouldShow is true when authenticated and not completed", () => {
    const { result } = renderHook(() => useOnboarding());
    expect(result.current.shouldShow).toBe(true);
  });

  it("shouldShow is false after complete()", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.complete());
    expect(result.current.shouldShow).toBe(false);
  });

  it("restart() resets completion", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.complete());
    expect(result.current.shouldShow).toBe(false);
    act(() => result.current.restart());
    expect(result.current.shouldShow).toBe(true);
  });

  // "Not now" and "don't show again" must not be the same decision: the first
  // has to be recoverable by simply coming back.
  it("dismissForSession() hides the tour without persisting completion", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.dismissForSession());

    expect(result.current.shouldShow).toBe(false);
    expect(localStorage.getItem("jobber-onboarding-completed")).toBeNull();
    expect(sessionStorage.getItem("jobber-onboarding-dismissed")).toBe("true");
  });

  it("a session dismissal is gone on the next visit", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.dismissForSession());
    expect(result.current.shouldShow).toBe(false);

    // A new browser session starts with empty sessionStorage.
    sessionStorage.clear();
    const next = renderHook(() => useOnboarding());
    expect(next.result.current.shouldShow).toBe(true);
  });

  it("complete() survives a new session", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.complete());

    sessionStorage.clear();
    const next = renderHook(() => useOnboarding());
    expect(next.result.current.shouldShow).toBe(false);
  });

  it("restart() clears a session dismissal too", () => {
    const { result } = renderHook(() => useOnboarding());
    act(() => result.current.dismissForSession());
    act(() => result.current.restart());

    expect(result.current.shouldShow).toBe(true);
  });
});

describe("useOnboardingHighlight", () => {
  it("returns null by default", () => {
    const { result } = renderHook(() => useOnboardingHighlight());
    expect(result.current).toBe(null);
  });

  it("updates when setOnboardingHighlight is called", () => {
    const { result } = renderHook(() => useOnboardingHighlight());
    act(() => setOnboardingHighlight("/app/companies"));
    expect(result.current).toBe("/app/companies");
  });

  it("returns null after clearing", () => {
    const { result } = renderHook(() => useOnboardingHighlight());
    act(() => setOnboardingHighlight("/app/companies"));
    act(() => setOnboardingHighlight(null));
    expect(result.current).toBe(null);
  });
});
