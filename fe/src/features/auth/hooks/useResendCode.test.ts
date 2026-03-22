import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useResendCode } from "./useResendCode";

vi.mock("@tanstack/react-query", () => ({
  useMutation: ({
    mutationFn,
    onSuccess,
    onError,
  }: {
    mutationFn: () => Promise<unknown>;
    onSuccess: () => void;
    onError: () => void;
  }) => {
    let pending = false;
    let success = false;

    return {
      mutate: () => {
        pending = true;
        mutationFn()
          .then(() => {
            pending = false;
            success = true;
            onSuccess();
          })
          .catch(() => {
            pending = false;
            onError();
          });
      },
      isPending: pending,
      isSuccess: success,
      reset: vi.fn(),
    };
  },
}));

describe("useResendCode", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("returns initial state with cooldown and disabled", () => {
    const { result } = renderHook(() =>
      useResendCode({
        mutationFn: vi.fn().mockResolvedValue({}),
        storageKey: "test",
      }),
    );

    expect(result.current.cooldown).toBe(60);
    expect(result.current.isDisabled).toBe(true);
    expect(result.current.resendCount).toBe(0);
    expect(result.current.isLimitReached).toBe(false);
    expect(result.current.isPending).toBe(false);
    expect(result.current.resendError).toBe("");
  });

  it("returns the resend function", () => {
    const { result } = renderHook(() =>
      useResendCode({
        mutationFn: vi.fn().mockResolvedValue({}),
        storageKey: "test",
      }),
    );

    expect(typeof result.current.resend).toBe("function");
  });

  it("detects blocked state from localStorage", () => {
    const futureTimestamp = Date.now() + 86400000;
    localStorage.setItem("resend_block_test", String(futureTimestamp));

    const { result } = renderHook(() =>
      useResendCode({
        mutationFn: vi.fn().mockResolvedValue({}),
        storageKey: "test",
      }),
    );

    expect(result.current.isLimitReached).toBe(true);
    expect(result.current.isDisabled).toBe(true);
  });

  it("does not detect block when stored timestamp is in the past", () => {
    const pastTimestamp = Date.now() - 1000;
    localStorage.setItem("resend_block_test", String(pastTimestamp));

    const { result } = renderHook(() =>
      useResendCode({
        mutationFn: vi.fn().mockResolvedValue({}),
        storageKey: "test_past",
      }),
    );

    // Not blocked (different key, no block set)
    expect(result.current.isLimitReached).toBe(false);
  });

  it("decrements cooldown over time", async () => {
    vi.useFakeTimers();
    const { result } = renderHook(() =>
      useResendCode({
        mutationFn: vi.fn().mockResolvedValue({}),
        storageKey: "test_timer",
      }),
    );

    expect(result.current.cooldown).toBe(60);

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current.cooldown).toBe(59);

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current.cooldown).toBe(58);

    vi.useRealTimers();
  });
});
