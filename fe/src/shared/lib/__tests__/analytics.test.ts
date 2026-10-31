import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import en from "@/shared/locales/en.json";

const trackers = vi.hoisted(() => ({
  trackGA4Event: vi.fn(),
  trackPostHogEvent: vi.fn(),
}));
const consent = vi.hoisted(() => ({
  getStoredConsent: vi.fn<() => "accepted" | "essential" | null>(),
}));

vi.mock("../gtag", () => ({ trackGA4Event: trackers.trackGA4Event }));
vi.mock("../posthog", () => ({
  trackPostHogEvent: trackers.trackPostHogEvent,
}));
vi.mock("../consent", () => consent);

import {
  BEGIN_CHECKOUT_DELIVERY_MS,
  PRO_PRICE_USD,
  trackBeginCheckout,
  trackConversion,
  type ConversionEvent,
} from "../analytics";

const EVERY_EVENT: readonly ConversionEvent[] = [
  { name: "sign_up" },
  { name: "email_verified" },
  { name: "purchase", transactionId: "checkout-1" },
];

const ALLOWED_PARAMS = new Set([
  "method",
  "currency",
  "value",
  "items",
  "transaction_id",
]);

beforeEach(() => {
  vi.resetAllMocks();
  consent.getStoredConsent.mockReturnValue("accepted");
});

describe("trackConversion", () => {
  it.each(EVERY_EVENT)("sends $name to both trackers", (event) => {
    trackConversion(event);

    expect(trackers.trackGA4Event).toHaveBeenCalledOnce();
    expect(trackers.trackGA4Event.mock.calls[0]?.[0]).toBe(event.name);
    expect(trackers.trackPostHogEvent).toHaveBeenCalledOnce();
    expect(trackers.trackPostHogEvent.mock.calls[0]?.[0]).toBe(event.name);
  });

  it.each(["essential", null] as const)(
    "sends nothing while consent is %s",
    (decision) => {
      consent.getStoredConsent.mockReturnValue(decision);

      EVERY_EVENT.forEach(trackConversion);

      expect(trackers.trackGA4Event).not.toHaveBeenCalled();
      expect(trackers.trackPostHogEvent).not.toHaveBeenCalled();
    },
  );

  // Consent is read per event: a visitor who withdraws in mid-session stops
  // producing events at once, without reloading the page.
  it("stops as soon as consent is withdrawn", () => {
    trackConversion({ name: "sign_up" });
    consent.getStoredConsent.mockReturnValue(null);
    trackConversion({ name: "email_verified" });

    expect(trackers.trackGA4Event).toHaveBeenCalledOnce();
  });

  it("carries only the fixed parameters, for every event", () => {
    EVERY_EVENT.forEach(trackConversion);

    for (const [, params] of trackers.trackGA4Event.mock.calls) {
      expect(
        Object.keys(params).filter((key) => !ALLOWED_PARAMS.has(key)),
      ).toEqual([]);
    }
  });

  it("names the checkout on the purchase, so GA4 can drop a repeat", () => {
    trackConversion({ name: "purchase", transactionId: "checkout-1" });

    expect(trackers.trackGA4Event.mock.calls[0]?.[1]).toMatchObject({
      transaction_id: "checkout-1",
      currency: "USD",
      value: PRO_PRICE_USD,
    });
  });

  it("never lets a tracker failure reach the caller", () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    trackers.trackGA4Event.mockImplementationOnce(() => {
      throw new Error("blocked");
    });

    expect(() => trackConversion({ name: "sign_up" })).not.toThrow();
    consoleError.mockRestore();
  });

  it("quotes the same list price the pricing page shows", () => {
    expect(en.settings.subscription.pricing.proPrice).toContain(
      `$${PRO_PRICE_USD}`,
    );
  });
});

describe("trackBeginCheckout", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  function settled(promise: Promise<void>): { done: boolean } {
    const state = { done: false };
    void promise.then(() => {
      state.done = true;
    });
    return state;
  }

  it("sends begin_checkout to both trackers, flushed for the redirect", () => {
    trackers.trackGA4Event.mockReturnValue(true);

    void trackBeginCheckout();

    expect(trackers.trackPostHogEvent.mock.calls[0]?.[0]).toBe(
      "begin_checkout",
    );
    expect(trackers.trackPostHogEvent.mock.calls[0]?.[2]).toBe(true);
    expect(trackers.trackGA4Event.mock.calls[0]?.[2]).toEqual(
      expect.any(Function),
    );
  });

  it("proceeds as soon as gtag has handed the event over", async () => {
    trackers.trackGA4Event.mockImplementation((_name, _params, delivered) => {
      delivered();
      return true;
    });

    await expect(trackBeginCheckout()).resolves.toBeUndefined();
  });

  // A blocked gtag.js never calls back, and a buyer must not be left waiting.
  it("proceeds on its own when gtag never answers", async () => {
    vi.useFakeTimers();
    trackers.trackGA4Event.mockReturnValue(true);

    const state = settled(trackBeginCheckout());
    await vi.advanceTimersByTimeAsync(BEGIN_CHECKOUT_DELIVERY_MS - 1);
    expect(state.done).toBe(false);

    await vi.advanceTimersByTimeAsync(1);
    expect(state.done).toBe(true);
  });

  it("does not wait when GA4 is not on the page", async () => {
    vi.useFakeTimers();
    trackers.trackGA4Event.mockReturnValue(false);

    const state = settled(trackBeginCheckout());
    await vi.advanceTimersByTimeAsync(0);

    expect(state.done).toBe(true);
  });

  it("does not wait, or send anything, without consent", async () => {
    vi.useFakeTimers();
    consent.getStoredConsent.mockReturnValue("essential");

    const state = settled(trackBeginCheckout());
    await vi.advanceTimersByTimeAsync(0);

    expect(state.done).toBe(true);
    expect(trackers.trackGA4Event).not.toHaveBeenCalled();
    expect(trackers.trackPostHogEvent).not.toHaveBeenCalled();
  });
});
