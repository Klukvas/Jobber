// The cookie decision the prerenderer seeds before it renders a route.
//
// Shared with the app on purpose: this module is the only place the seeded
// record is written, and `scripts/__tests__/prerenderConsent.test.mjs` feeds it
// to the app's own `getStoredConsent` to prove the app still honours it. That
// test is the guard — a record the app rejects is treated as no decision at
// all, which bakes the cookie banner into every prerendered snapshot and lets
// analytics-shaped code run during builds.

/** localStorage key `src/shared/lib/consent.ts` reads the decision from. */
export const PRERENDER_CONSENT_KEY = "cookie-consent";

/**
 * A refusal, deliberately carrying no policy version.
 *
 * The version used to be copied here as a literal `1`, which is a second copy
 * of a constant that lives in `consent.ts` and drifts silently the moment that
 * one is bumped. There is nothing to drift now: a refusal is honoured whatever
 * policy it was given against — re-asking somebody who declined is nagging, not
 * consent — so the seed simply does not name one.
 */
export const PRERENDER_CONSENT_RECORD = Object.freeze({
  choice: "essential",
  at: "prerender",
});
