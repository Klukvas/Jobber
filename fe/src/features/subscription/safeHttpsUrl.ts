/**
 * The normalized URL when `raw` is safe to send the browser to, else null.
 *
 * Backend responses are external data headed for `location.assign`: only an
 * absolute `https:` URL with a host is accepted, so `javascript:`, `data:` and
 * relative values never get through. Userinfo is refused too —
 * `https://creem.io@evil.com` reads as Creem in a glance at the address but
 * lands on evil.com.
 */
export function safeHttpsUrl(raw: unknown): string | null {
  if (typeof raw !== "string") return null;
  try {
    const url = new URL(raw);
    const isSafe =
      url.protocol === "https:" &&
      url.hostname !== "" &&
      url.username === "" &&
      url.password === "";
    return isSafe ? url.href : null;
  } catch {
    return null;
  }
}

/**
 * Hosts the billing provider's checkout and customer portal are served from.
 *
 * Without it any https URL the backend returned would be followed. To be
 * confirmed against the real checkout/portal hosts in the first test-mode
 * purchase (checklist in docs/adr/0003-creem-billing.md).
 */
export const BILLING_HOST_SUFFIX = "creem.io";

const MAX_BILLING_URL_LENGTH = 2048;

/** Space, ASCII control characters and backslash — all of which the URL parser silently repairs. */
function hasRepairableCharacters(raw: string): boolean {
  for (const char of raw) {
    const code = char.charCodeAt(0);
    if (code <= 0x20 || code === 0x7f || char === "\\") return true;
  }
  return false;
}

function isBillingHost(hostname: string): boolean {
  const host = hostname.toLowerCase();
  return (
    host === BILLING_HOST_SUFFIX || host.endsWith(`.${BILLING_HOST_SUFFIX}`)
  );
}

/**
 * The URL when `raw` is a well-formed https link on the billing provider's
 * domain, else null.
 *
 * Stricter than `safeHttpsUrl` because the WHATWG parser normalises junk such
 * as `https:\\evil.test` or a leading space into a valid URL: the raw string
 * is checked first so nothing is "fixed" into acceptance.
 */
export function safeBillingUrl(raw: unknown): string | null {
  if (typeof raw !== "string") return null;
  if (raw.length > MAX_BILLING_URL_LENGTH || hasRepairableCharacters(raw)) {
    return null;
  }
  // `https:/host` and `https:host` are also repaired into `https://host`.
  if (!/^https:\/\//i.test(raw)) return null;

  const normalized = safeHttpsUrl(raw);
  if (!normalized) return null;

  const url = new URL(normalized);
  if (url.port !== "" || !isBillingHost(url.hostname)) return null;
  return normalized;
}
