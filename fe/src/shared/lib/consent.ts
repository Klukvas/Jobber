import { initGA4 } from "./gtag";
import { initPostHog } from "./posthog";

const STORAGE_KEY = "cookie-consent";

export type CookieConsent = "accepted" | "essential";

export function getStoredConsent(): CookieConsent | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return value === "accepted" || value === "essential" ? value : null;
  } catch {
    return null;
  }
}

// Analytics (GA4, PostHog) load ONLY after explicit opt-in — the site must be
// fully usable without tracking cookies. Essential cookies (auth, Paddle
// checkout) are not gated: the service cannot function without them.
export function applyConsent(consent: CookieConsent): void {
  try {
    localStorage.setItem(STORAGE_KEY, consent);
  } catch {
    // Private mode: the banner reappears next visit — acceptable.
  }
  if (consent === "accepted") {
    initGA4();
    initPostHog();
  }
}

export function initAnalyticsIfConsented(): void {
  if (getStoredConsent() === "accepted") {
    initGA4();
    initPostHog();
  }
}
