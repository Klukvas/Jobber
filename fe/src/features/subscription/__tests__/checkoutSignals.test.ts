import { describe, it, expect, vi, beforeEach } from "vitest";

import {
  PRE_CHECKOUT_PLAN_KEY,
  PRE_CHECKOUT_TTL_MS,
  readFreshPreCheckoutPlan,
  forgetPreCheckoutPlan,
  notifyCheckoutCompleted,
  onCheckoutCompleted,
  readPreCheckoutPlan,
  rememberPreCheckoutPlan,
} from "../checkoutSignals";

describe("the pre-checkout baseline", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("survives a reload, which is the only reason it is on disk", () => {
    rememberPreCheckoutPlan("pro");
    // Asserted through the reader: the stored shape carries a timestamp, and
    // pinning the serialization here would only make it hard to change.
    expect(readPreCheckoutPlan()).toBe("pro");
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).not.toBeNull();
    expect(readPreCheckoutPlan()).toBe("pro");
  });

  it("reads as free when no checkout was ever started", () => {
    expect(readPreCheckoutPlan()).toBe("free");
  });

  it("is forgotten so a later page load does not wait for an upgrade", () => {
    rememberPreCheckoutPlan("pro");
    forgetPreCheckoutPlan();

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    expect(readPreCheckoutPlan()).toBe("free");
  });
});

describe("the checkout-completed signal", () => {
  it("reaches a listener on the same page load", () => {
    // The popup never navigates, so this event is the layout's only cue.
    const listener = vi.fn();
    const unsubscribe = onCheckoutCompleted(listener);

    notifyCheckoutCompleted();

    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("stops reaching a listener that unsubscribed", () => {
    const listener = vi.fn();
    onCheckoutCompleted(listener)();

    notifyCheckoutCompleted();

    expect(listener).not.toHaveBeenCalled();
  });
});

/**
 * A page load has to decide whether to believe a key it did not write. The
 * checkout popup can end without the provider saying so, and a leftover
 * baseline is what turned that silence into a full-screen "Activating your
 * subscription…" for someone who never paid.
 */
describe("readFreshPreCheckoutPlan", () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.useRealTimers();
  });

  it("is null when no checkout was started", () => {
    expect(readFreshPreCheckoutPlan()).toBeNull();
  });

  it("returns a baseline recorded just now", () => {
    rememberPreCheckoutPlan("pro");
    expect(readFreshPreCheckoutPlan()).toBe("pro");
  });

  it("ignores a baseline older than the window", () => {
    vi.useFakeTimers();
    rememberPreCheckoutPlan("pro");
    vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS + 1);

    expect(readFreshPreCheckoutPlan()).toBeNull();
    vi.useRealTimers();
  });

  // Left in place, a stale entry would re-arm the overlay on every navigation.
  it("drops an expired baseline from storage as it reads it", () => {
    vi.useFakeTimers();
    rememberPreCheckoutPlan("pro");
    vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS + 1);

    readFreshPreCheckoutPlan();

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
    vi.useRealTimers();
  });

  it("keeps a baseline that is still inside the window", () => {
    vi.useFakeTimers();
    rememberPreCheckoutPlan("pro");
    vi.advanceTimersByTime(PRE_CHECKOUT_TTL_MS - 1_000);

    expect(readFreshPreCheckoutPlan()).toBe("pro");
    vi.useRealTimers();
  });

  /**
   * An entry with no timestamp came from a build that did not write one, which
   * means it came from an earlier page load — this one always stamps. Nothing
   * distinguishes a checkout started a minute ago from one abandoned last
   * week, and "no timestamp" used to mean "never expires": the entry stayed
   * fresh for the whole session and re-armed the full-screen "activating your
   * subscription" on every navigation, for somebody who never paid.
   */
  it("treats an undated entry as expired on mount", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "pro");

    expect(readFreshPreCheckoutPlan()).toBeNull();
  });

  it("clears the undated entry so it cannot keep re-arming the overlay", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "pro");

    readFreshPreCheckoutPlan();

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  /**
   * "How long ago" is computed as `now - at`, which goes *negative* for a
   * timestamp in the future — and a negative age is never greater than the
   * window, so such an entry never expired. A clock the user set forward and
   * then corrected leaves one behind, and sessionStorage is writable by
   * anything on the origin, so one can also simply be planted. Either way the
   * result was a permanent full-screen "activating your subscription".
   */
  it("rejects a baseline dated beyond any plausible clock skew", () => {
    sessionStorage.setItem(
      PRE_CHECKOUT_PLAN_KEY,
      JSON.stringify({ plan: "pro", at: Date.now() + 24 * 60 * 60_000 }),
    );

    expect(readFreshPreCheckoutPlan()).toBeNull();
  });

  it("drops the future-dated entry rather than leaving it to try again", () => {
    sessionStorage.setItem(
      PRE_CHECKOUT_PLAN_KEY,
      JSON.stringify({ plan: "pro", at: Date.now() + 24 * 60 * 60_000 }),
    );

    readFreshPreCheckoutPlan();

    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  // Two clocks are never exactly equal, and a checkout that really did just
  // start must survive the difference.
  it("still accepts a baseline a moment ahead of this clock", () => {
    sessionStorage.setItem(
      PRE_CHECKOUT_PLAN_KEY,
      JSON.stringify({ plan: "pro", at: Date.now() + 1_000 }),
    );

    expect(readFreshPreCheckoutPlan()).toBe("pro");
  });

  // The in-page completion event has already seen the checkout happen and only
  // needs the plan, so it is not subject to the freshness rule — but the entry
  // still has to be a plan the app recognises.
  it("still answers the in-page reader from a valid current-session entry", () => {
    rememberPreCheckoutPlan("pro");

    expect(readPreCheckoutPlan()).toBe("pro");
  });

  it("ignores a corrupted entry", () => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, "{not json");
    expect(readFreshPreCheckoutPlan()).toBeNull();
  });

  it.each([
    ["truncated json", "{not json"],
    ["json that is not an object", '"pro"'],
    ["a plan nobody ships", JSON.stringify({ plan: "platinum", at: Date.now() })],
    ["a legacy string naming no known plan", "platinum"],
    ["a plan of the wrong type", JSON.stringify({ plan: 7, at: Date.now() })],
    ["no plan at all", JSON.stringify({ at: Date.now() })],
  ])("deletes %s rather than keeping it around", (_name, raw) => {
    sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, raw);

    expect(readFreshPreCheckoutPlan()).toBeNull();
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBeNull();
  });

  // An unrecognised plan cannot be ranked against the backend's, so it must not
  // reach the comparison at all — it falls back to the safe floor instead.
  it("falls back to free for an unknown stored plan", () => {
    sessionStorage.setItem(
      PRE_CHECKOUT_PLAN_KEY,
      JSON.stringify({ plan: "platinum", at: Date.now() }),
    );

    expect(readPreCheckoutPlan()).toBe("free");
  });

  it("keeps every plan the app does ship", () => {
    for (const plan of ["free", "pro", "enterprise"] as const) {
      rememberPreCheckoutPlan(plan);
      expect(readFreshPreCheckoutPlan()).toBe(plan);
    }
  });
});

/**
 * Storage is a convenience here, never a precondition. Safari's private mode
 * throws on write, and a full quota throws on any origin — neither may be able
 * to stop a purchase from starting or leave the caller stuck.
 */
describe("storage failures are survivable", () => {
  it("does not throw when the baseline cannot be written", () => {
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new DOMException("quota exceeded", "QuotaExceededError");
      });

    try {
      expect(() => rememberPreCheckoutPlan("pro")).not.toThrow();
    } finally {
      setItem.mockRestore();
    }
  });

  it("does not throw when the baseline cannot be cleared", () => {
    const removeItem = vi
      .spyOn(Storage.prototype, "removeItem")
      .mockImplementation(() => {
        throw new DOMException("denied", "SecurityError");
      });

    try {
      expect(() => forgetPreCheckoutPlan()).not.toThrow();
    } finally {
      removeItem.mockRestore();
    }
  });

  it("reads as 'nothing pending' when storage cannot be read", () => {
    const getItem = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new DOMException("denied", "SecurityError");
      });

    try {
      expect(readFreshPreCheckoutPlan()).toBeNull();
      expect(readPreCheckoutPlan()).toBe("free");
    } finally {
      getItem.mockRestore();
    }
  });
});
