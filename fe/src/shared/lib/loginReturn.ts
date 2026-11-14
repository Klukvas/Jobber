/**
 * Where to send someone after they sign in, carried on the login URL.
 *
 * A protected page that finds no session used to send its visitor to the
 * landing page, with no login form and no memory of where they were going.
 * The login URL now names the page instead, and the login form goes back to
 * it. Only in-app paths are accepted: the value comes from the address bar, so
 * anything else (another origin, `//host`) would be an open redirect.
 */
export const RETURN_PARAM = "next";

const IN_APP_PATH = /^\/app(?:[/?#]|$)/;

function isInAppPath(value: string | null | undefined): value is string {
  return typeof value === "string" && IN_APP_PATH.test(value);
}

export function loginPathFor(returnTo?: string | null): string {
  if (!isInAppPath(returnTo)) return "/login";
  return `/login?${RETURN_PARAM}=${encodeURIComponent(returnTo)}`;
}

export function readReturnTo(search: string): string | null {
  const value = new URLSearchParams(search).get(RETURN_PARAM);
  return isInAppPath(value) ? value : null;
}
