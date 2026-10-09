import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useCheckout } from "../useCheckout";
import {
  PRE_CHECKOUT_PLAN_KEY,
  readFreshPreCheckoutPlan,
} from "../checkoutSignals";

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

const mockConfig = vi.hoisted(() => ({ value: undefined as unknown }));
const mockGetQueryData = vi.hoisted(() => vi.fn());
const mockCreateSession = vi.hoisted(() => vi.fn());

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

const readyConfig = {
  provider: "creem",
  environment: "test",
  plans: ["pro", "enterprise"],
};

const CHECKOUT_URL = "https://www.creem.io/test/checkout/prod_1/ch_1";

describe("useCheckout", () => {
  let assignSpy: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockCreateSession.mockResolvedValue({ checkout_url: CHECKOUT_URL });
    mockGetQueryData.mockReturnValue({ plan: "free" });
    sessionStorage.clear();

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

  it("reports ready when the backend advertises plans", () => {
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

  it("redirects to the checkout URL the backend returned", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(mockCreateSession).toHaveBeenCalledWith("pro");
    expect(assignSpy).toHaveBeenCalledTimes(1);
    expect(assignSpy).toHaveBeenCalledWith(CHECKOUT_URL);
    expect(result.current.error).toBeNull();
  });

  it("persists the baseline plan before redirecting", async () => {
    mockGetQueryData.mockReturnValue({ plan: "pro" });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(readFreshPreCheckoutPlan()).toBe("pro");
  });

  it("defaults the baseline to free when no subscription is cached", async () => {
    mockGetQueryData.mockReturnValue(undefined);
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(readFreshPreCheckoutPlan()).toBe("free");
  });

  it("stays pending after the redirect starts", async () => {
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(result.current.isPending).toBe(true);
  });

  it("starts only one checkout on a double click", async () => {
    let release: (value: unknown) => void = () => {};
    mockCreateSession.mockReturnValue(
      new Promise((resolve) => {
        release = resolve;
      }),
    );
    const { result } = renderHook(() => useCheckout());

    await act(async () => {
      void result.current.openCheckout("pro");
      void result.current.openCheckout("pro");
    });
    await act(async () => release({ checkout_url: CHECKOUT_URL }));

    expect(mockCreateSession).toHaveBeenCalledTimes(1);
    expect(assignSpy).toHaveBeenCalledTimes(1);
  });

  it("does not start a checkout for a plan the backend does not sell", async () => {
    mockConfig.value = { ...readyConfig, plans: ["pro"] };
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("enterprise"));

    expect(mockCreateSession).not.toHaveBeenCalled();
    expect(assignSpy).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  describe.each([
    ["an http URL", { checkout_url: "http://www.creem.io/checkout/1" }],
    ["a javascript: URL", { checkout_url: "javascript:alert(1)" }],
    ["a data: URL", { checkout_url: "data:text/html,<script>1</script>" }],
    ["a relative URL", { checkout_url: "/checkout/1" }],
    ["a protocol-relative URL", { checkout_url: "//evil.example/checkout" }],
    ["a malformed URL", { checkout_url: "https://" }],
    ["a URL with a username", { checkout_url: "https://creem.io@evil.com/x" }],
    ["a URL with credentials", { checkout_url: "https://user:pw@creem.io/x" }],
    ["a blank URL", { checkout_url: "" }],
    ["a non-string URL", { checkout_url: 42 }],
    ["an empty body", {}],
    ["no body", undefined],
  ])("when the backend returns %s", (_label, response) => {
    beforeEach(() => {
      mockCreateSession.mockResolvedValue(response);
    });

    it("never navigates and surfaces the error", async () => {
      const { result } = renderHook(() => useCheckout());

      await act(() => result.current.openCheckout("pro"));

      expect(assignSpy).not.toHaveBeenCalled();
      expect(result.current.error?.message).toBe(
        "The checkout could not be completed",
      );
      expect(result.current.isPending).toBe(false);
      expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    });
  });

  it("clears the baseline and surfaces the error when the session fails", async () => {
    mockCreateSession.mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(result.current.error?.message).toBe("boom");
    expect(result.current.isPending).toBe(false);
    expect(assignSpy).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  it("retries after a failure instead of staying stuck", async () => {
    mockCreateSession.mockRejectedValueOnce(new Error("boom"));
    const { result } = renderHook(() => useCheckout());
    await act(() => result.current.openCheckout("pro"));

    await act(() => result.current.openCheckout("pro"));

    expect(assignSpy).toHaveBeenCalledWith(CHECKOUT_URL);
    expect(result.current.error).toBeNull();
  });

  it("recovers and reports when the pre-redirect work throws", async () => {
    mockGetQueryData.mockImplementation(() => {
      throw new Error("cache exploded");
    });
    const { result } = renderHook(() => useCheckout());

    await act(() => result.current.openCheckout("pro"));

    expect(result.current.error?.message).toBe("cache exploded");
    expect(result.current.isPending).toBe(false);
    expect(assignSpy).not.toHaveBeenCalled();
  });

  describe("when sessionStorage refuses the baseline write", () => {
    it.each(["QuotaExceededError", "SecurityError"])(
      "still redirects on %s",
      async (name) => {
        const setItem = vi
          .spyOn(Storage.prototype, "setItem")
          .mockImplementation(() => {
            throw new DOMException(name, name);
          });
        try {
          const { result } = renderHook(() => useCheckout());

          await act(() => result.current.openCheckout("pro"));

          expect(assignSpy).toHaveBeenCalledWith(CHECKOUT_URL);
          expect(result.current.error).toBeNull();
        } finally {
          setItem.mockRestore();
        }
      },
    );
  });

  describe("pageshow", () => {
    function firePageShow(persisted: boolean) {
      const event = new Event("pageshow") as PageTransitionEvent;
      Object.defineProperty(event, "persisted", { value: persisted });
      act(() => {
        window.dispatchEvent(event);
      });
    }

    it("resets pending when the page is restored from the bfcache", async () => {
      const { result } = renderHook(() => useCheckout());
      await act(() => result.current.openCheckout("pro"));
      expect(result.current.isPending).toBe(true);

      firePageShow(true);

      expect(result.current.isPending).toBe(false);
      await act(() => result.current.openCheckout("pro"));
      expect(assignSpy).toHaveBeenCalledTimes(2);
    });

    it("keeps pending on an ordinary page show", async () => {
      const { result } = renderHook(() => useCheckout());
      await act(() => result.current.openCheckout("pro"));

      firePageShow(false);

      expect(result.current.isPending).toBe(true);
    });
  });
});
