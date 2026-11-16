/** Safe destinations carried through the three auth routes. */
export const RETURN_PARAM = "next";
const CLOSE_PARAM = "returnTo";
export type AuthPath = "/login" | "/register" | "/forgot-password";

function safePath(value: string | null | undefined): value is string {
  if (!value || !value.startsWith("/") || value.startsWith("//") || (value.includes("\\") || Array.from(value).some((char) => char.charCodeAt(0) <= 32 || char.charCodeAt(0) === 127))) return false;
  try {
    const url = new URL(value, "https://cursopus.local");
    return url.origin === "https://cursopus.local";
  } catch { return false; }
}

function isInAppPath(value: string | null | undefined): value is string {
  return safePath(value) && /^\/app(?:\/|$)/.test(new URL(value, "https://cursopus.local").pathname);
}

export function readReturnTo(search: string): string | null {
  const value = new URLSearchParams(search).get(RETURN_PARAM);
  return isInAppPath(value) ? value : null;
}

/** Closing returns only to public reading context, independently of post-login next. */
export function readAuthCloseReturn(search: string): string {
  const value = new URLSearchParams(search).get(CLOSE_PARAM);
  if (!safePath(value)) return "/";
  const pathname = new URL(value, "https://cursopus.local").pathname;
  return pathname === "/" || /^\/features\/(applications|resume-builder|cover-letters)\/?$/.test(pathname) ? value : "/";
}

export function authPathFor(path: AuthPath, search = "", closeReturn?: string): string {
  const params = new URLSearchParams();
  const next = readReturnTo(search);
  if (next) params.set(RETURN_PARAM, next);
  const returnTo = readAuthCloseReturn(closeReturn ? `?${CLOSE_PARAM}=${encodeURIComponent(closeReturn)}` : search);
  if (returnTo !== "/") params.set(CLOSE_PARAM, returnTo);
  return path + (params.size ? `?${params}` : "");
}

export function loginPathFor(returnTo?: string | null): string {
  return authPathFor("/login", isInAppPath(returnTo) ? `?next=${encodeURIComponent(returnTo)}` : "");
}
