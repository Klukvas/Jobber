/**
 * The two signals that connect "a checkout was started" to "the app should
 * watch for the upgrade": one for the same page load, one for the next one.
 *
 * They live in their own module because both ends need them — the checkout hook
 * that starts a purchase and the layout that reacts to it — and neither should
 * have to import the other.
 */

import type { SubscriptionPlan } from "@/shared/types/api";

/**
 * The plan the user was on when the checkout popup opened.
 *
 * It is a *baseline*, not a claim: success is only ever shown once the backend
 * reports a strictly higher plan than this. It lives in sessionStorage rather
 * than component state so a reload mid-purchase — which destroys the popup with
 * the page — still comes back watching for the upgrade.
 */
export const PRE_CHECKOUT_PLAN_KEY = "billing_pre_checkout_plan";

/**
 * Fired when the popup closes reporting an order. The popup never navigates, so
 * without this the layout would have no way to notice the purchase until the
 * next page load.
 */
const CHECKOUT_COMPLETED_EVENT = "jobber:checkout-completed";

/** Records the plan the user is on before the popup opens. */
export function rememberPreCheckoutPlan(plan: SubscriptionPlan): void {
  sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, plan);
}

/**
 * Forgets the baseline. Called whenever a checkout ends without a purchase, so
 * the next page load does not sit polling for an upgrade that cannot arrive.
 */
export function forgetPreCheckoutPlan(): void {
  sessionStorage.removeItem(PRE_CHECKOUT_PLAN_KEY);
}

/** Reads the baseline, defaulting to free when nothing was recorded. */
export function readPreCheckoutPlan(): SubscriptionPlan {
  return (
    (sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY) as SubscriptionPlan | null) ??
    "free"
  );
}

/**
 * Announces that the popup closed on an order.
 *
 * This says "start watching", nothing more. The provider's callback is not
 * evidence of payment — the webhook is — so the listener polls the backend and
 * only celebrates once the plan there has actually moved.
 */
export function notifyCheckoutCompleted(): void {
  window.dispatchEvent(new CustomEvent(CHECKOUT_COMPLETED_EVENT));
}

/** Subscribes to checkout completion. Returns the unsubscribe function. */
export function onCheckoutCompleted(listener: () => void): () => void {
  window.addEventListener(CHECKOUT_COMPLETED_EVENT, listener);
  return () => window.removeEventListener(CHECKOUT_COMPLETED_EVENT, listener);
}
