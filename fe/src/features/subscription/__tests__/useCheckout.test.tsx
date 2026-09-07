import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

import { useCheckout } from "../useCheckout";
import { PRE_CHECKOUT_PLAN_KEY } from "../checkoutSignals";
import type { PopupCheckoutHandlers } from "../fastspringSbl";

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

const mockConfig = vi.hoisted(() => ({ value: undefined as unknown }));
const mockGetQueryData = vi.hoisted(() => vi.fn());
const mockCreateSession = vi.hoisted(() => vi.fn());
const mockOpenPopup = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: mockConfig.value }),
  useQueryClient: () => ({ getQueryData: mockGetQueryData }),
  // Minimal stand-in: calls the mutation function directly.
  useMutation: ({ mutationFn }: { mutationFn: (plan: string) => unknown }) => ({
    mutateAsync: mutationFn,
    isPending: false,
  }),
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: {
    getCheckoutConfig: vi.fn(),
    createCheckoutSession: (plan: string) => mockCreateSession(plan),
  },
}));

vi.mock("@/features/subscription/fastspringSbl", () => ({
  openPopupCheckout: (options: unknown) => mockOpenPopup(options),
}));

const readyConfig = {
  provider: "fastspring",
  environment: "test",
  storefront: "fluxlab.test.onfastspring.com/popup-jobber",
  plans: ["pro", "enterprise"],
};

/** The handlers the hook handed to the popup on the last open. */
function popupHandlers(): PopupCheckoutHandlers {
  const options = mockOpenPopup.mock.calls.at(-1)?.[0] as {
    handlers: PopupCheckoutHandlers;
  };
  return options.handlers;
}

describe("useCheckout", () => {
  let assignSpy: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockCreateSession.mockResolvedValue({ session_id: "sess-1" });
    mockOpenPopup.mockResolvedValue(undefined);
    mockGetQueryData.mockReturnValue({ plan: "free" });
    sessionStorage.clear();

    // The whole point of the popup is that nothing navigates. Any assignment
    // here would be the full-page checkout coming back.
    assignSpy = vi.fn();
    originalLocation = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, assign: assignSpy },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
    sessionStorage.clear();
  });

  it("reports ready when the backend advertises plans and a storefront", () => {
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(true);
  });

  it("reports not ready when there is no config", () => {
    mockConfig.value = undefined;
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("reports not ready when the backend advertises no plans", () => {
    mockConfig.value = { ...readyConfig, plans: [] };
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("reports not ready when the backend advertises no storefront", () => {
    // An unopenable checkout must not be offered at all.
    mockConfig.value = { ...readyConfig, storefront: "" };
    const { result } = renderHook(() => useCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("opens the popup on this page instead of navigating anywhere", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockCreateSession).toHaveBeenCalledWith("pro");
    expect(mockOpenPopup).toHaveBeenCalledTimes(1);
    expect(mockOpenPopup.mock.calls[0][0]).toMatchObject({
      storefront: readyConfig.storefront,
      environment: "test",
      sessionId: "sess-1",
    });
    expect(assignSpy).not.toHaveBeenCalled();
  });

  it("passes the backend's session id and no checkout URL", async () => {
    // The session id is the entire cart; a URL is something to navigate to.
    mockCreateSession.mockResolvedValue({
      session_id: "sess-9",
      expires_at: "2026-01-01T00:00:00Z",
    });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    const options = mockOpenPopup.mock.calls[0][0] as Record<string, unknown>;
    expect(options.sessionId).toBe("sess-9");
    expect(Object.keys(options).sort()).toEqual([
      "environment",
      "handlers",
      "sessionId",
      "storefront",
    ]);
  });

  it("sends no user identity to the provider from the browser", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    // The plan is the only thing the browser gets to choose; the buyer comes
    // from the authenticated session on the backend.
    expect(mockCreateSession).toHaveBeenCalledTimes(1);
    expect(mockCreateSession.mock.calls[0]).toEqual(["pro"]);
  });

  it("persists the baseline plan before the popup opens", async () => {
    mockGetQueryData.mockReturnValue({ plan: "pro" });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("pro");
  });

  it("defaults the baseline to free when no subscription is cached", async () => {
    mockGetQueryData.mockReturnValue(undefined);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("free");
  });

  it("stays pending while the popup is open", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    expect(result.current.isPending).toBe(true);

    await act(async () => popupHandlers().onClose({ completed: true }));
    expect(result.current.isPending).toBe(false);
  });

  it("ignores a second click while a checkout is already starting", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(async () => {
      await Promise.all([
        result.current.openCheckout("pro"),
        result.current.openCheckout("pro"),
      ]);
    });

    expect(mockCreateSession).toHaveBeenCalledTimes(1);
    expect(mockOpenPopup).toHaveBeenCalledTimes(1);
  });

  it("announces a completed purchase without claiming it as one", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onClose({ completed: true }));

    // The event only starts polling — the baseline stays for the layout to read.
    expect(completed).toHaveBeenCalledTimes(1);
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("free");
    expect(result.current.error).toBeNull();
    window.removeEventListener("jobber:checkout-completed", completed);
  });

  it("clears the baseline and celebrates nothing when the buyer cancels", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onClose({ completed: false }));

    expect(completed).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(result.current.isPending).toBe(false);
    expect(result.current.error).toBeNull();
    window.removeEventListener("jobber:checkout-completed", completed);
  });

  it("surfaces a provider error and clears the baseline", async () => {
    const completed = vi.fn();
    window.addEventListener("jobber:checkout-completed", completed);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await act(async () => popupHandlers().onError());

    expect(completed).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(result.current.isPending).toBe(false);
    await waitFor(() => expect(result.current.error).not.toBeNull());
  });

  it("retries after a failure instead of staying stuck", async () => {
    mockOpenPopup.mockRejectedValueOnce(new Error("script blocked"));
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));
    await waitFor(() =>
      expect(result.current.error?.message).toBe("script blocked"),
    );
    expect(result.current.isPending).toBe(false);
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();

    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).toHaveBeenCalledTimes(2);
    expect(result.current.error).toBeNull();
  });

  it("does not start a checkout for a plan the backend does not sell", async () => {
    mockConfig.value = { ...readyConfig, plans: ["pro"] };
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(mockCreateSession).not.toHaveBeenCalled();
    expect(mockOpenPopup).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  it("does not start a checkout with no storefront to open it in", async () => {
    mockConfig.value = { ...readyConfig, storefront: "" };
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockCreateSession).not.toHaveBeenCalled();
    expect(mockOpenPopup).not.toHaveBeenCalled();
  });

  it("clears the baseline and surfaces the error when the session fails", async () => {
    mockCreateSession.mockRejectedValue(new Error("checkout unavailable"));
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockOpenPopup).not.toHaveBeenCalled();
    expect(assignSpy).not.toHaveBeenCalled();
    // A stale baseline would make the next page load poll for an upgrade that
    // can never arrive.
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    await waitFor(() =>
      expect(result.current.error?.message).toBe("checkout unavailable"),
    );
  });
});
