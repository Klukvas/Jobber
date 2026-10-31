import { getStoredConsent } from "./consent";
import { trackGA4Event } from "./gtag";
import { trackPostHogEvent } from "./posthog";

/**
 * List price of Pro, the only plan on sale. The amount actually paid can differ
 * after a discount, so revenue is read from Creem and this only sizes the funnel.
 */
export const PRO_PRICE_USD = 24;

/**
 * How long the checkout redirect waits for gtag to hand `begin_checkout` over.
 * gtag sends on its own schedule, so an immediate redirect loses the hit, but a
 * buyer must not be kept waiting by a blocked or slow script.
 */
export const BEGIN_CHECKOUT_DELIVERY_MS = 400;

const PRO_ITEM = {
  item_id: "pro",
  item_name: "Cursopus Pro",
  price: PRO_PRICE_USD,
  quantity: 1,
};

const PRO_PURCHASE = {
  currency: "USD",
  value: PRO_PRICE_USD,
  items: [PRO_ITEM],
};

/**
 * The conversions, in GA4's recommended names where one exists.
 *
 * A closed list rather than a name and a free-form object, so what can leave
 * the browser is decided here: no call site can pass an email, a job title or
 * any text a customer wrote.
 */
export type ConversionEvent =
  | { readonly name: "sign_up" }
  | { readonly name: "email_verified" }
  | { readonly name: "begin_checkout" }
  | { readonly name: "purchase"; readonly transactionId: string };

function paramsFor(event: ConversionEvent): Readonly<Record<string, unknown>> {
  switch (event.name) {
    case "sign_up":
      return { method: "email" };
    case "email_verified":
      return {};
    case "begin_checkout":
      return PRO_PURCHASE;
    case "purchase":
      return { ...PRO_PURCHASE, transaction_id: event.transactionId };
  }
}

/**
 * Reports a conversion to every tracker the visitor has allowed, and answers
 * whether GA4 took it.
 *
 * Called from the sign-up and checkout paths, so it must never be the reason one
 * of them fails: an event lost is a number off in a report, nothing more.
 */
function send(event: ConversionEvent, onDelivered?: () => void): boolean {
  if (getStoredConsent() !== "accepted") return false;

  const params = paramsFor(event);
  try {
    trackPostHogEvent(event.name, params, onDelivered !== undefined);
    return trackGA4Event(event.name, params, onDelivered);
  } catch (error) {
    console.error("Analytics event failed", event.name, error);
    return false;
  }
}

export function trackConversion(event: ConversionEvent): void {
  send(event);
}

/**
 * Reports the start of a checkout, resolving when the redirect may go ahead: as
 * soon as GA4 has the event, after BEGIN_CHECKOUT_DELIVERY_MS at the latest, and
 * at once when nothing was sent, so a visitor without consent never waits.
 */
export function trackBeginCheckout(): Promise<void> {
  return new Promise((proceed) => {
    const sent = send({ name: "begin_checkout" }, proceed);
    if (sent) {
      setTimeout(proceed, BEGIN_CHECKOUT_DELIVERY_MS);
    } else {
      proceed();
    }
  });
}
