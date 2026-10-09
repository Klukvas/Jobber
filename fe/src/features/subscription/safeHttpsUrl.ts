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
