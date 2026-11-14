import { z } from "zod";
import { readAuthorizationParams } from "@/services/oauthService";

export const CHECKOUT_CONSENT_KEY = "oauth_checkout_consent";
export const CHECKOUT_CONSENT_TTL_MS = 30 * 60_000;
const storedConsentSchema = z.object({ path: z.string(), at: z.number().finite() }).strict();

function isConsentPath(path: string): boolean {
  if (!path.startsWith("/oauth/authorize?") || /[\s\\#]/u.test(path)) return false;
  try {
    const url = new URL(path, window.location.origin);
    if (url.origin !== window.location.origin || url.pathname !== "/oauth/authorize") return false;
    const params = readAuthorizationParams(url.search);
    // Store only authorization inputs, never an arbitrary query containing codes or tokens.
    return [...url.searchParams.keys()].every((key) => Object.hasOwn(params, key)) &&
      params.response_type === "code" && !!params.client_id && !!params.redirect_uri &&
      !!params.code_challenge && params.code_challenge_method === "S256";
  } catch {
    return false;
  }
}

export function forgetCheckoutConsent(): void {
  try {
    sessionStorage.removeItem(CHECKOUT_CONSENT_KEY);
  } catch {
    // Storage may be disabled. Resuming consent is best-effort.
  }
}

/** Called only when checkout starts, so another purchase cannot reuse stale consent. */
export function rememberCheckoutConsent(path: string): void {
  forgetCheckoutConsent();
  if (!isConsentPath(path)) return;
  try {
    sessionStorage.setItem(CHECKOUT_CONSENT_KEY, JSON.stringify({ path, at: Date.now() }));
  } catch {
    // Checkout remains usable when browser storage is unavailable.
  }
}

/** Consumed once after a confirmed upgrade; the destination re-runs backend preview. */
export function takeCheckoutConsent(): string | null {
  try {
    const raw = sessionStorage.getItem(CHECKOUT_CONSENT_KEY);
    sessionStorage.removeItem(CHECKOUT_CONSENT_KEY);
    if (!raw) return null;
    const parsed = storedConsentSchema.safeParse(JSON.parse(raw));
    if (!parsed.success) return null;
    const age = Date.now() - parsed.data.at;
    return age >= 0 && age < CHECKOUT_CONSENT_TTL_MS && isConsentPath(parsed.data.path)
      ? parsed.data.path : null;
  } catch {
    // Refuse corrupt or inaccessible storage without blocking Settings.
    return null;
  }
}
