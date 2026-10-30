import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import {
  CANCEL_POLL_INTERVAL_MS,
  CANCEL_POLL_TIMEOUT_MS,
  useCancellationWatch,
} from "../useCancellationWatch";
import type { SubscriptionDTO } from "@/shared/types/api";

const getSubscription = vi.hoisted(() => vi.fn());

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: { getSubscription },
}));

const CANCEL_AT = "2026-11-30T00:00:00Z";

function subscriptionRow(overrides: Partial<SubscriptionDTO> = {}) {
  return {
    plan: "pro",
    status: "active",
    current_period_end: CANCEL_AT,
    ...overrides,
  } as SubscriptionDTO;
}

// The real useSubscription over a real client: what is under test is whether a
// webhook arriving after the request ever reaches the cache.
async function renderWatch() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const view = renderHook(() => useCancellationWatch(), { wrapper });
  await elapse(0);
  return view;
}

// react-query hands a result to its observers through nested zero-delay timers,
// and fake time counts a zero delay set during a tick as a millisecond.
const NOTIFY_MS = 2;

const elapse = (ms: number) =>
  act(() => vi.advanceTimersByTimeAsync(ms + NOTIFY_MS));

describe("useCancellationWatch", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    getSubscription.mockReset();
    getSubscription.mockResolvedValue(subscriptionRow());
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not poll until a cancellation is accepted", async () => {
    const { result } = await renderWatch();

    await elapse(CANCEL_POLL_INTERVAL_MS * 3);

    expect(result.current.isCancelPending).toBe(false);
    expect(getSubscription).toHaveBeenCalledOnce();
  });

  it("picks up a webhook that lands after the request returned", async () => {
    const { result } = await renderWatch();

    act(() => result.current.watchCancellation());
    expect(result.current.isCancelPending).toBe(true);

    await elapse(CANCEL_POLL_INTERVAL_MS);
    expect(result.current.isCancelPending).toBe(true);

    getSubscription.mockResolvedValue(
      subscriptionRow({ cancel_at: CANCEL_AT }),
    );
    await elapse(CANCEL_POLL_INTERVAL_MS);

    expect(result.current.isCancelPending).toBe(false);
  });

  it("stops polling as soon as the row shows the cancellation", async () => {
    const { result } = await renderWatch();
    getSubscription.mockResolvedValue(
      subscriptionRow({ cancel_at: CANCEL_AT }),
    );

    act(() => result.current.watchCancellation());
    await elapse(CANCEL_POLL_INTERVAL_MS);
    const readsWhenConfirmed = getSubscription.mock.calls.length;

    await elapse(CANCEL_POLL_INTERVAL_MS * 5);

    expect(getSubscription.mock.calls.length).toBe(readsWhenConfirmed);
  });

  // A cancellation can end the subscription outright, and then there is no
  // `cancel_at` to wait for.
  it("treats a plan that went free as confirmed", async () => {
    const { result } = await renderWatch();

    act(() => result.current.watchCancellation());
    getSubscription.mockResolvedValue(
      subscriptionRow({ plan: "free", status: "cancelled" }),
    );
    await elapse(CANCEL_POLL_INTERVAL_MS);

    expect(result.current.isCancelPending).toBe(false);
  });

  it("gives up polling at the timeout if the webhook never arrives", async () => {
    const { result } = await renderWatch();

    act(() => result.current.watchCancellation());
    await elapse(CANCEL_POLL_TIMEOUT_MS);
    const readsAtTimeout = getSubscription.mock.calls.length;

    await elapse(CANCEL_POLL_INTERVAL_MS * 5);

    expect(getSubscription.mock.calls.length).toBe(readsAtTimeout);
  });

  // The provider has the cancellation whether or not the webhook made it, so
  // the row's "renews on" is still not to be believed.
  it("keeps the cancellation pending after polling gave up", async () => {
    const { result } = await renderWatch();

    act(() => result.current.watchCancellation());
    await elapse(CANCEL_POLL_TIMEOUT_MS + CANCEL_POLL_INTERVAL_MS);

    expect(result.current.isCancelPending).toBe(true);
  });

  it("stops polling when the page goes away", async () => {
    const { result, unmount } = await renderWatch();

    act(() => result.current.watchCancellation());
    await elapse(CANCEL_POLL_INTERVAL_MS);
    const readsAtUnmount = getSubscription.mock.calls.length;

    unmount();
    await elapse(CANCEL_POLL_INTERVAL_MS * 5);

    expect(getSubscription.mock.calls.length).toBe(readsAtUnmount);
  });
});
