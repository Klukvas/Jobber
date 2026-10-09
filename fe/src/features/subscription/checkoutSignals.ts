/**
 * The pre-checkout baseline that connects "a checkout was started" to "the app
 * should watch for the upgrade" after the redirect back.
 *
 * It lives in its own module because both ends need it — the checkout hook
 * that starts a purchase and the layout that reacts to it — and neither should
 * have to import the other.
 */

import type { SubscriptionPlan } from "@/shared/types/api";

/**
 * The plan the user was on when the checkout redirect started.
 *
 * It is a *baseline*, not a claim: success is only ever shown once the backend
 * reports a strictly higher plan than this. It lives in sessionStorage rather
 * than component state so a reload mid-purchase — which loses the page — still comes back watching for the upgrade.
 */
export const PRE_CHECKOUT_PLAN_KEY = "billing_pre_checkout_plan";

/**
 * How long a baseline is worth acting on.
 *
 * A checkout that is still going after this has either finished — in which
 * case the backend already knows and the plan simply reads higher — or been
 * abandoned. Acting on an older entry only produces the thing this window
 * exists to prevent: a full-screen "activating your subscription" shown to
 * somebody who never paid. Generous enough to cover a slow card form, short
 * enough that a forgotten tab does not greet its owner with a spinner.
 */
export const PRE_CHECKOUT_TTL_MS = 30 * 60_000;

/**
 * How far ahead of this clock a baseline may be stamped and still be believed.
 *
 * Freshness is `now - at`, which is *negative* for a future timestamp — and a
 * negative age is never greater than the window, so an entry dated ahead never
 * expired at all. That is not hypothetical: a device whose clock was running
 * fast when the checkout started, and correct by the time the page reloaded,
 * writes exactly one. sessionStorage is also writable by anything on the
 * origin, so one can simply be planted. Either way it produced a permanent
 * full-screen "activating your subscription" for somebody who never paid.
 *
 * A minute is enough for the ordinary case — two clocks are never exactly
 * equal — and far short of anything that could be mistaken for a real wait.
 */
export const PRE_CHECKOUT_MAX_CLOCK_SKEW_MS = 60_000;

interface StoredBaseline {
  readonly plan: SubscriptionPlan;
  /** Epoch milliseconds. Absent on entries written by older builds. */
  readonly at?: number;
}

/**
 * Every plan the app knows how to compare. sessionStorage is writable by
 * anything running on the origin, and the value ends up in a "did the plan go
 * up?" comparison, so an unrecognised string is thrown away rather than
 * carried forward as a plan nobody can rank.
 *
 * Typed as the plan union rather than as `string[]`, so a plan added to the
 * API's type and forgotten here is a compile error rather than a value this
 * guard silently rejects at runtime.
 */
const KNOWN_PLANS = [
  "free",
  "pro",
  "enterprise",
] as const satisfies readonly SubscriptionPlan[];

function isKnownPlan(value: unknown): value is SubscriptionPlan {
  return (
    typeof value === "string" &&
    (KNOWN_PLANS as readonly string[]).includes(value)
  );
}

/** Parses whatever is in storage, tolerating the older plain-string format. */
function parseBaseline(raw: string | null): StoredBaseline | null {
  if (!raw) return null;
  if (!raw.startsWith("{")) {
    return isKnownPlan(raw) ? { plan: raw } : null;
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    const plan = (parsed as { plan?: unknown })?.plan;
    if (!isKnownPlan(plan)) return null;
    const at = (parsed as { at?: unknown })?.at;
    return {
      plan,
      at: typeof at === "number" && Number.isFinite(at) ? at : undefined,
    };
  } catch {
    return null;
  }
}

/**
 * Reads the baseline and clears the key when what is there cannot be used.
 *
 * Junk is deleted rather than ignored: a value that fails to parse fails to
 * parse on every navigation, and leaving it in place means the next write —
 * the one that would have fixed it — is the only thing that ever can.
 */
function takeUsableBaseline(): StoredBaseline | null {
  let raw: string | null = null;
  try {
    raw = sessionStorage.getItem(PRE_CHECKOUT_PLAN_KEY);
  } catch {
    return null;
  }
  if (raw === null) return null;

  const baseline = parseBaseline(raw);
  if (!baseline) {
    forgetPreCheckoutPlan();
    return null;
  }
  return baseline;
}

/**
 * Records the plan the user is on before the checkout redirect, and when.
 *
 * Best-effort on purpose. This is read by the *next* page load; a
 * storage jar that refuses writes — Safari's private mode, a full quota — must
 * not be able to stop a purchase from starting. Losing it costs the
 * "activating" overlay after the redirect back, nothing more.
 */
export function rememberPreCheckoutPlan(plan: SubscriptionPlan): void {
  try {
    sessionStorage.setItem(
      PRE_CHECKOUT_PLAN_KEY,
      JSON.stringify({ plan, at: Date.now() }),
    );
  } catch {
    // Nothing recorded; the plan change is still picked up by normal refetches.
  }
}

/**
 * Forgets the baseline. Called whenever a checkout ends without a purchase, so
 * the next page load does not sit polling for an upgrade that cannot arrive.
 */
export function forgetPreCheckoutPlan(): void {
  try {
    sessionStorage.removeItem(PRE_CHECKOUT_PLAN_KEY);
  } catch {
    // Nothing was stored to forget.
  }
}

/**
 * The baseline a fresh page load should act on, or null when there is none
 * worth acting on.
 *
 * A mount has to decide whether to believe a key it did not write, not just
 * read the plan. Expired entries are dropped from storage as they are
 * read, so a stale one cannot keep re-arming the overlay on every navigation.
 */
export function readFreshPreCheckoutPlan(): SubscriptionPlan | null {
  const baseline = takeUsableBaseline();
  if (!baseline) return null;

  if (baseline.at === undefined) {
    // Written by a build that did not timestamp it, which means it was written
    // by an *earlier page load* — this one always stamps. There is no way to
    // tell a checkout that started a minute ago from one abandoned last week,
    // and treating it as fresh made every undated entry permanently fresh:
    // one that lived through the whole session, re-arming the full-screen
    // "activating your subscription" on every navigation for somebody who
    // never paid. Expired is the safe reading, and it self-heals on the next
    // real checkout.
    forgetPreCheckoutPlan();
    return null;
  }

  const age = Date.now() - baseline.at;
  if (age > PRE_CHECKOUT_TTL_MS || age < -PRE_CHECKOUT_MAX_CLOCK_SKEW_MS) {
    forgetPreCheckoutPlan();
    return null;
  }
  return baseline.plan;
}
