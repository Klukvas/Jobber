import posthog from "posthog-js";
import { initGA4, setGA4Disabled, trackGA4PageView } from "./gtag";
import { initPostHog, trackPageView } from "./posthog";

const STORAGE_KEY = "cookie-consent";

export const CONSENT_RESET_EVENT = "jobber:cookie-consent-reset";

/**
 * Which version of the cookie policy a stored decision was given against.
 *
 * The record has always carried a timestamp "so we can re-ask after future
 * policy changes", but nothing said which policy the visitor had actually been
 * shown — so there was no way to re-ask, and a decision made against the first
 * banner would have stood for a policy it never described. Consent has to be
 * informed, and a decision about a different document is not.
 *
 * Bump this whenever the banner's description of what is collected changes.
 * Everyone whose *acceptance* names an older version (or none, which is every
 * record written before this existed) is treated as undecided: the banner comes
 * back and nothing loads until they answer it again. A *refusal* is not
 * re-asked — see `getStoredConsent`.
 */
export const CONSENT_POLICY_VERSION = 1;

export type CookieConsent = "accepted" | "essential";

let analyticsStarted = false;
let entryPageviewTracked = false;

/**
 * Whether this build can load any analytics at all.
 *
 * Both trackers are opt-in per environment and are usually unconfigured, in
 * which case accepting cookies loads nothing. The consent copy has to say so:
 * describing collection that cannot happen overstates what we do.
 */
export function isAnalyticsConfigured(): boolean {
  const ga4 =
    import.meta.env.VITE_FEATURE_GA4 === "true" &&
    !!import.meta.env.VITE_GA4_MEASUREMENT_ID;
  const posthogConfigured =
    import.meta.env.VITE_FEATURE_POSTHOG === "true" &&
    !!import.meta.env.VITE_POSTHOG_KEY;
  return ga4 || posthogConfigured;
}

interface StoredDecision {
  readonly choice: CookieConsent;
  /** Absent on the plain-string and versionless formats earlier builds wrote. */
  readonly version?: number;
}

/**
 * Whatever is in storage, in the shape this build understands.
 *
 * Three formats have shipped: a bare `"accepted"` / `"essential"` string, an
 * object with no `version`, and the current stamped object. The first two say
 * what the visitor chose but not which policy they were shown, so they are
 * parsed here and judged by `getStoredConsent` rather than discarded outright.
 */
function parseStoredDecision(): StoredDecision | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    if (raw === "accepted" || raw === "essential") return { choice: raw };

    const parsed = JSON.parse(raw) as { choice?: unknown; version?: unknown };
    const choice = parsed?.choice;
    if (choice !== "accepted" && choice !== "essential") return null;
    const version = parsed?.version;
    return typeof version === "number" ? { choice, version } : { choice };
  } catch {
    return null;
  }
}

/**
 * The visitor's decision as this build should act on it, or null when there is
 * none to honour and the banner has to ask.
 *
 * The two answers are not symmetrical, and treating them as if they were is
 * what this fixes.
 *
 * An **acceptance** is permission, and permission has to be informed: a yes
 * given to a policy that has since changed was a yes to a different document,
 * so it does not carry over. Neither does one from a format that cannot name
 * the policy at all — the plain string, or the versionless object. Those go
 * back to the banner, and nothing starts until it is answered again.
 *
 * A **refusal** is the withdrawal of permission, and nothing about a rewritten
 * policy turns "do not track me" back into an open question. Discarding it made
 * the banner reappear for everybody who had already said no — on every policy
 * bump, and permanently for the older formats — which is exactly the nagging a
 * stored refusal exists to end. It is honoured whatever version it names.
 *
 * Unreadable and unrecognised values stay null: undecided is the only safe
 * direction to be wrong in.
 */
export function getStoredConsent(): CookieConsent | null {
  const decision = parseStoredDecision();
  if (!decision) return null;
  if (decision.choice === "essential") return "essential";
  return decision.version === CONSENT_POLICY_VERSION ? "accepted" : null;
}

// Analytics (GA4, PostHog) load ONLY after explicit opt-in — the site must be
// fully usable without tracking cookies. Essential cookies (authentication) are
// not gated: the service cannot function without them. Checkout happens on the
// Creem hosted checkout, which sets its own cookies on its own domain.
export function applyConsent(consent: CookieConsent): void {
  try {
    // Timestamped so we can show when consent was given, and stamped with the
    // policy it was given against so a later change can re-ask rather than
    // assume the answer still applies.
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        choice: consent,
        at: new Date().toISOString(),
        version: CONSENT_POLICY_VERSION,
      }),
    );
  } catch {
    // Private mode: the banner reappears next visit — acceptable.
  }

  if (consent === "accepted") {
    startAnalytics();
    // The router already skipped tracking this page while consent was
    // undecided — record the current page once or the entry pageview is
    // lost. Same relative format as usePageTracking.
    if (!entryPageviewTracked) {
      entryPageviewTracked = true;
      const url = window.location.pathname + window.location.search;
      trackGA4PageView(url);
      trackPageView(url);
    }
  } else {
    stopAnalytics();
  }
}

export function initAnalyticsIfConsented(): void {
  const consent = getStoredConsent();
  if (consent === "accepted") {
    startAnalytics();
    return;
  }
  // "essential" is a decision this build honours, whichever policy it was
  // given against. Nothing to start, and nothing to clean up — the withdrawal
  // that produced it already ran.
  if (consent === "essential") return;

  // Past here the stored value is one this build will not honour — an
  // acceptance of an older policy, or a record it cannot read. The banner is
  // about to ask again, and until it is answered nothing may be running: an
  // "accepted" that has stopped counting must not leave its identifiers alive.
  // Cleared as well as swept, so this happens once rather than on every load.
  if (hasStoredDecision()) {
    dropStoredConsent();
    stopAnalytics();
  }
}

/** Whether anything at all is recorded, honoured by this build or not. */
function hasStoredDecision(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) !== null;
  } catch {
    return false;
  }
}

/** Forgets the stored decision. The banner reappears on the next read. */
function dropStoredConsent(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing stored — the banner will show anyway.
  }
}

// Withdrawing consent must be as easy as giving it (GDPR art. 7(3)): the
// footer "Cookie settings" link calls this — the stored choice is dropped,
// analytics stop, and the banner re-opens. No sign-out involved.
export function resetConsent(): void {
  dropStoredConsent();
  stopAnalytics();
  window.dispatchEvent(new Event(CONSENT_RESET_EVENT));
}

function startAnalytics(): void {
  // A withdrawal that is being undone must not have its cleanup land on the
  // identity the new consent is about to create.
  cancelWithdrawalSweeps();

  // Lift the GA kill switch from a possible earlier in-session withdrawal.
  setGA4Disabled(false);

  if (!analyticsStarted) {
    analyticsStarted = true;
    initGA4();
    initPostHog();
  }

  // Give PostHog its storage back before opting in — a client left with
  // persistence disabled would forget the visitor on every page load.
  setPostHogPersistence(true);

  // PostHog PERSISTS its opt-out in its own storage across sessions — without
  // this, a user who once opted out and later accepts would have their
  // explicit consent silently ignored forever. captureEventName: null skips
  // the synthetic $opt_in event.
  if (posthog.__loaded && posthog.has_opted_out_capturing()) {
    posthog.opt_in_capturing({ captureEventName: null });
  }
}

/**
 * PostHog's own per-project storage keys, in both the cookie jar and
 * localStorage: `ph_<project-key>_posthog` and siblings such as
 * `ph_<project-key>_posthog_rate_limit`.
 *
 * Anchored at both ends deliberately. It must not reach `__ph_opt_in_out_*`,
 * which is the *choice* rather than the identity: deleting that would let the
 * next `posthog.init()` treat the visitor as undecided and start capturing
 * again — the exact thing withdrawal exists to prevent.
 */
const POSTHOG_STORAGE_KEY = /^ph_.+_posthog(_[A-Za-z0-9_]+)?$/;

/**
 * Cookie names withdrawal has to clear.
 *
 * Both vendors are first-party here, so the identifiers they leave behind are
 * ours to expire — and they must be. Stopping collection while keeping the id
 * means the same person is re-identified the moment consent is granted again,
 * which is not what "essential only" promises.
 *
 * Every pattern is anchored at both ends: an unanchored `_gat` also matched
 * `_gateway`, and an unanchored `ph_` prefix would have reached storage that
 * has nothing to do with analytics.
 */
const ANALYTICS_COOKIE_PATTERNS: readonly RegExp[] = [
  /^_ga$/, // GA4 client id
  /^_ga_[A-Za-z0-9_-]+$/, // GA4 per-property session state
  /^_gid$/, // legacy GA session id
  /^_gat(_[A-Za-z0-9_-]+)?$/, // GA throttling flag, incl. _gat_gtag_* and _gat_UA-*
  POSTHOG_STORAGE_KEY, // PostHog per-key store, incl. its rate-limit sibling
];

function isAnalyticsCookie(name: string): boolean {
  return ANALYTICS_COOKIE_PATTERNS.some((pattern) => pattern.test(name));
}

/**
 * Every path a cookie could have been scoped to, from the current URL upwards.
 *
 * A cookie is only deleted by a write that matches its own path and domain, and
 * the browser never tells us which pair was used. GA writes at `/`, but a
 * cookie set while the customer was deeper in the app would survive a `/`-only
 * expiry — which is exactly how `_ga` and `ph_*_posthog` were still present
 * after withdrawal.
 */
export function cookiePathVariants(pathname: string): string[] {
  const paths = ["/"];
  const segments = pathname.split("/").filter(Boolean);
  let current = "";
  for (const segment of segments) {
    current += `/${segment}`;
    paths.push(current);
  }
  return paths;
}

/**
 * Every domain a first-party cookie could have been scoped to: the host itself
 * (host-only, written with no `domain` attribute) and each registrable parent.
 * The bare TLD is skipped — no browser accepts it — and single-label hosts like
 * `localhost` only ever get the host-only form.
 *
 * No public-suffix list is consulted, and none is needed. A `domain=` write is
 * only honoured for a suffix of the writing document's own host, and the
 * browser refuses any value that is itself a public suffix — so an attempt at
 * `co.uk` from `shop.co.uk` is a no-op, not a way to reach anyone else's
 * cookies. Over-listing costs a rejected write; under-listing leaves the
 * identifier in place, which is the failure that matters here.
 */
export function cookieDomainVariants(hostname: string): (string | null)[] {
  const domains: (string | null)[] = [null];
  const labels = hostname.split(".");
  if (labels.length < 2) return domains;

  for (let i = 0; i <= labels.length - 2; i += 1) {
    const domain = labels.slice(i).join(".");
    domains.push(domain, `.${domain}`);
  }
  return domains;
}

/** Expires one cookie across every path/domain pair it might have been set on. */
function expireCookie(name: string, pathname: string, hostname: string): void {
  const expiry = "expires=Thu, 01 Jan 1970 00:00:00 GMT";
  for (const path of cookiePathVariants(pathname)) {
    for (const domain of cookieDomainVariants(hostname)) {
      const scope = domain ? `; domain=${domain}` : "";
      document.cookie = `${name}=; ${expiry}; path=${path}${scope}`;
    }
  }
}

/**
 * Clears the identifiers analytics left behind. Exported for tests; withdrawal
 * always goes through `applyConsent`.
 */
export function clearAnalyticsCookies(): void {
  try {
    const { hostname, pathname } = window.location;
    for (const entry of document.cookie.split(";")) {
      const name = entry.split("=")[0]?.trim();
      if (!name || !isAnalyticsCookie(name)) continue;
      expireCookie(name, pathname, hostname);
    }
  } catch {
    // Cookie cleanup is best-effort: a blocked cookie jar must not break
    // the rest of the withdrawal.
  }
}

/**
 * Drops PostHog's identity out of localStorage.
 *
 * `posthog.reset()` only does this when the library is actually on the page,
 * and it usually is not: the script is loaded lazily, so a visitor who accepted
 * on an earlier visit and withdraws on this one leaves `ph_<key>_posthog` — the
 * distinct id and its session — sitting in storage with nothing to clear it.
 * The next accepted visit would then pick the same person straight back up.
 *
 * The opt-out flag (`__ph_opt_in_out_<key>`) is deliberately left alone; see
 * POSTHOG_STORAGE_KEY.
 */
export function clearPostHogStorage(): void {
  try {
    const doomed = Object.keys(localStorage).filter((key) =>
      POSTHOG_STORAGE_KEY.test(key),
    );
    for (const key of doomed) {
      localStorage.removeItem(key);
    }
  } catch {
    // Storage denied or unavailable: nothing was persisted to clear either.
  }
}

/**
 * Turns PostHog's own writes to localStorage and the cookie jar on or off.
 *
 * Withdrawal used to be a race it kept losing. `reset()` and the storage sweep
 * below run in one synchronous burst, but the library writes its store from
 * places we never call — a `/flags` response that was already on the wire lands
 * about 150 ms later and saves the props back out. That is how "essential only"
 * ended with `ph_<key>_posthog` present again, carrying a brand new distinct
 * id, and surviving a reload: capture was off, but the identity was not.
 *
 * `disable_persistence` is the door rather than the mop. Once it is set, every
 * later save is a no-op and the store is removed from both backends, so there
 * is no window left for a late writer to land in. The opt-out flag lives under
 * `__ph_opt_in_out_<key>` and is kept by a separate mechanism, so it survives
 * this untouched — the withdrawal still outlives the page.
 */
function setPostHogPersistence(enabled: boolean): void {
  if (!posthog.__loaded) return;
  posthog.set_config({ disable_persistence: !enabled });
}

/**
 * When to re-sweep PostHog's storage after a withdrawal.
 *
 * Disabling persistence stops future writes, but it cannot un-write one that
 * landed a tick earlier — and it does nothing at all when the library is not on
 * the page yet. These passes cover that window and then stop: withdrawal must
 * not leave a timer running for the rest of the session. The 150 ms mark is the
 * one the regression actually reproduced at; the rest bracket it.
 */
const WITHDRAWAL_SWEEP_DELAYS_MS: readonly number[] = [
  50, 150, 400, 1_000, 3_000,
];

let withdrawalSweeps: readonly number[] = [];

function cancelWithdrawalSweeps(): void {
  for (const timer of withdrawalSweeps) {
    window.clearTimeout(timer);
  }
  withdrawalSweeps = [];
}

function scheduleWithdrawalSweeps(): void {
  cancelWithdrawalSweeps();
  withdrawalSweeps = WITHDRAWAL_SWEEP_DELAYS_MS.map((delay) =>
    window.setTimeout(() => {
      clearPostHogStorage();
      clearAnalyticsCookies();
    }, delay),
  );
}

// Cleanup for state set while consent was granted. Both vendors are stopped
// first — a live client would simply write its cookies back — and only then are
// the identifiers expired.
function stopAnalytics(): void {
  // An already-loaded gtag.js keeps sending router pageviews (and re-setting
  // the _ga cookies we just expired) unless Google's kill switch is set.
  setGA4Disabled(true);

  if (posthog.__loaded) {
    // Order matters, and it is the opposite of what it looks like. reset()
    // clears PostHog's persistence and starts a fresh anonymous identity —
    // including the stored opt-in/out decision — so calling it *after*
    // opt_out_capturing() threw the opt-out away and left the client capturing
    // again under a new id. Drop the identity first, then set the flag that has
    // to outlive it.
    posthog.reset();
    posthog.opt_out_capturing();
    // Last, so nothing the two calls above scheduled can still reach storage.
    setPostHogPersistence(false);
  }

  // Runs whether or not the library ever loaded: the identity outlives the
  // page that wrote it.
  clearPostHogStorage();
  clearAnalyticsCookies();
  scheduleWithdrawalSweeps();
}
