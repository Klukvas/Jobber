import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import {
  usePaddleCheckout,
  PRE_CHECKOUT_PLAN_KEY,
} from "../usePaddleCheckout";

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { PAYMENTS: true },
}));

const mockConfig = vi.hoisted(() => ({
  value: undefined as unknown,
}));
const mockGetQueryData = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: mockConfig.value }),
  useQueryClient: () => ({ getQueryData: mockGetQueryData }),
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: { getCheckoutConfig: vi.fn() },
}));

const mockUser = vi.hoisted(() => ({
  value: { id: "u1", email: "user@example.com" } as
    | { id: string; email: string }
    | null,
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: unknown) => unknown) =>
    selector({ user: mockUser.value }),
}));

const readyConfig = {
  client_token: "tok_123",
  environment: "production",
  prices: { pro: "price_pro", enterprise: "price_ent" },
};

describe("usePaddleCheckout", () => {
  let openSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    mockConfig.value = readyConfig;
    mockUser.value = { id: "u1", email: "user@example.com" };
    openSpy = vi.fn();
    (window as unknown as { Paddle?: unknown }).Paddle = {
      Initialize: vi.fn(),
      Environment: { set: vi.fn() },
      Checkout: { open: openSpy },
    };
    sessionStorage.clear();
  });

  afterEach(() => {
    delete (window as unknown as { Paddle?: unknown }).Paddle;
  });

  it("reports ready when a token and prices are present", () => {
    const { result } = renderHook(() => usePaddleCheckout());
    expect(result.current.isReady).toBe(true);
  });

  it("reports not ready when there is no config", () => {
    mockConfig.value = undefined;
    const { result } = renderHook(() => usePaddleCheckout());
    expect(result.current.isReady).toBe(false);
  });

  it("opens Paddle checkout with the selected plan price and customer email", () => {
    mockGetQueryData.mockReturnValue({ plan: "free" });
    const { result } = renderHook(() => usePaddleCheckout());

    result.current.openCheckout("pro");

    expect(openSpy).toHaveBeenCalledTimes(1);
    const opts = openSpy.mock.calls[0][0];
    expect(opts.items).toEqual([{ priceId: "price_pro", quantity: 1 }]);
    expect(opts.customer).toEqual({ email: "user@example.com" });
    expect(opts.customData).toEqual({ user_id: "u1" });
    expect(opts.settings.successUrl).toContain("subscription=success");
  });

  it("persists the baseline plan to sessionStorage before opening", () => {
    mockGetQueryData.mockReturnValue({ plan: "pro" });
    const { result } = renderHook(() => usePaddleCheckout());

    result.current.openCheckout("enterprise");

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("pro");
  });

  it("defaults the baseline to free when no subscription is cached", () => {
    mockGetQueryData.mockReturnValue(undefined);
    const { result } = renderHook(() => usePaddleCheckout());

    result.current.openCheckout("pro");

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("free");
  });

  it("does not open checkout when the price for the plan is missing", () => {
    mockConfig.value = { ...readyConfig, prices: {} };
    const { result } = renderHook(() => usePaddleCheckout());

    result.current.openCheckout("pro");

    expect(openSpy).not.toHaveBeenCalled();
  });

  it("omits customer/customData when there is no user", () => {
    mockUser.value = null;
    mockGetQueryData.mockReturnValue({ plan: "free" });
    const { result } = renderHook(() => usePaddleCheckout());

    result.current.openCheckout("pro");

    const opts = openSpy.mock.calls[0][0];
    expect(opts.customer).toBeUndefined();
    expect(opts.customData).toBeUndefined();
  });
});
