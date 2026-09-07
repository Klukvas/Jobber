import { describe, it, expect, vi, beforeEach } from "vitest";

import {
  PRE_CHECKOUT_PLAN_KEY,
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
    expect(sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY)).toBe("pro");
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
