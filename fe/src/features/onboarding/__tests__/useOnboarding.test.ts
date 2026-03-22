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
