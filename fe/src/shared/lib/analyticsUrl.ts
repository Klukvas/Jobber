/**
 * What an address may look like once it leaves the browser.
 *
 * Addresses here carry things that must not reach a third party. A reset link
 * lands on `/reset-password?email=&code=`, the jobs list keeps what was typed
 * into its search in `?search=`, and `/s/:token` puts a share token in the path.
 * Everything that reports a URL (GA4, PostHog, Sentry) goes through this one
 * function, so a new route only has to be safe here.
 */

const CAMPAIGN_PARAMS = [
  "utm_source",
  "utm_medium",
  "utm_campaign",
  "utm_term",
  "utm_content",
] as const;

/** Campaign tags are labels. Anything else in one is not worth keeping. */
const CAMPAIGN_VALUE = /^[\w.-]{1,64}$/;

/** Public paths such as `/blog/:slug` stay readable; these are private. */
const PRIVATE_PATHS: readonly (readonly [RegExp, string])[] = [
  [/^\/s\/[^/]+/, "/s/:token"],
  [
    /^\/app\/(jobs|applications|resume-builder|cover-letters)\/[^/]+/,
    "/app/$1/:id",
  ],
  [/^\/api\/v1\/public\/shares\/[^/]+/, "/api/v1/public/shares/:token"],
];

const ABSOLUTE_URL = /^[a-z][a-z\d+.-]*:/i;

export function cleanPath(pathname: string): string {
  return PRIVATE_PATHS.reduce(
    (path, [pattern, replacement]) => path.replace(pattern, replacement),
    pathname,
  );
}

function cleanSearch(params: URLSearchParams): string {
  const kept = CAMPAIGN_PARAMS.flatMap((name) => {
    const value = params.get(name);
    return value !== null && CAMPAIGN_VALUE.test(value)
      ? [`${name}=${value}`]
      : [];
  });
  return kept.length > 0 ? `?${kept.join("&")}` : "";
}

/**
 * The address with everything private removed, in the form it was given:
 * absolute stays absolute, relative stays relative. Another site's address is
 * reduced to its origin, and what cannot be read as a web address is dropped.
 */
export function cleanUrl(raw: string): string {
  if (!raw) return "";

  const { origin } = window.location;
  let url: URL;
  try {
    url = new URL(raw, origin);
  } catch {
    return "";
  }
  if (url.protocol !== "https:" && url.protocol !== "http:") return "";
  if (url.origin !== origin) return url.origin;

  const cleaned = `${cleanPath(url.pathname)}${cleanSearch(url.searchParams)}`;
  return ABSOLUTE_URL.test(raw) ? `${origin}${cleaned}` : cleaned;
}
