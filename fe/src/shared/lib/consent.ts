import posthog from "posthog-js";
import { initGA4, trackGA4PageView } from "./gtag";
import { initPostHog, trackPageView } from "./posthog";

const STORAGE_KEY = "cookie-consent";

export const CONSENT_RESET_EVENT = "jobber:cookie-consent-reset";

export type CookieConsent = "accepted" | "essential";

let analyticsStarted = false;

export function getStoredConsent(): CookieConsent | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    // Tolerate the early plain-string format.
    if (raw === "accepted" || raw === "essential") return raw;
    const parsed: unknown = JSON.parse(raw);
    const choice = (parsed as { choice?: unknown }).choice;
    return choice === "accepted" || choice === "essential" ? choice : null;
  } catch {
    return null;
  }
}

// Analytics (GA4, PostHog) load ONLY after explicit opt-in — the site must be
// fully usable without tracking cookies. Essential cookies (auth, Paddle
// checkout) are not gated: the service cannot function without them.
export function applyConsent(consent: CookieConsent): void {
  try {
    // Timestamped so we can show when consent was given and re-ask after
    // future policy changes.
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ choice: consent, at: new Date().toISOString() }),
    );
  } catch {
    // Private mode: the banner reappears next visit — acceptable.
  }

  if (consent === "accepted") {
    startAnalytics();
    // The router already skipped tracking this page while consent was
    // undecided — record the current page or the entry pageview is lost.
    trackGA4PageView(window.location.pathname + window.location.search);
    trackPageView(window.location.href);
  } else {
    stopAnalytics();
  }
}

export function initAnalyticsIfConsented(): void {
  if (getStoredConsent() === "accepted") {
    startAnalytics();
  }
}

// Withdrawing consent must be as easy as giving it (GDPR art. 7(3)): the
// footer "Cookie settings" link calls this — the stored choice is dropped,
// analytics stop, and the banner re-opens. No sign-out involved.
export function resetConsent(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing stored — the banner will show anyway.
  }
  stopAnalytics();
  window.dispatchEvent(new Event(CONSENT_RESET_EVENT));
}

function startAnalytics(): void {
  if (analyticsStarted) {
    // Re-accept after an in-session opt-out: the scripts are already loaded,
    // only PostHog's opt-out flag needs lifting.
    if (posthog.__loaded) {
      posthog.opt_in_capturing();
    }
    return;
  }
  analyticsStarted = true;
  initGA4();
  initPostHog();
}

// Best-effort cleanup for cookies set while consent was granted: GA cookies
// are first-party (_ga, _ga_*) so we can expire them ourselves; PostHog
// honors its own opt-out flag.
function stopAnalytics(): void {
  if (posthog.__loaded) {
    posthog.opt_out_capturing();
  }
  try {
    const hostname = window.location.hostname;
    document.cookie.split(";").forEach((entry) => {
      const name = entry.split("=")[0]?.trim();
      if (!name || !name.startsWith("_ga")) return;
      const expiry = "expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
      document.cookie = `${name}=; ${expiry}`;
      document.cookie = `${name}=; ${expiry}; domain=${hostname}`;
      document.cookie = `${name}=; ${expiry}; domain=.${hostname}`;
    });
  } catch {
    // Cookie cleanup is best-effort.
  }
}
