/**
 * The three URLs that draw an auth modal over the landing page.
 *
 * The landing page renders all three modals and picks one from the path, and
 * the layout that hosts those routes has to send an already-signed-in visitor
 * away from exactly the same set. One table, so the two cannot disagree about
 * what an auth route is.
 */

export type AuthModalRoute = "login" | "register" | "forgot-password";

const MODAL_BY_PATH: Readonly<Record<string, AuthModalRoute>> = {
  "/login": "login",
  "/register": "register",
  "/forgot-password": "forgot-password",
};

/**
 * The pathname as this table spells it.
 *
 * The router matches `/login` and `/login/` to the same route, so both render
 * the landing page — but an exact map read saw only the first, and the
 * trailing-slash form arrived at a landing page with no modal on it. A
 * signed-in visitor was not sent away from it either, because the layout asks
 * this same question. Both forms are one URL; they get one answer.
 */
function normalizePath(pathname: string): string {
  const trimmed = pathname.replace(/\/+$/, "");
  return trimmed === "" ? "/" : trimmed;
}

/** The modal a path opens, or null when the path is not an auth route. */
export function getAuthModalRoute(pathname: string): AuthModalRoute | null {
  return MODAL_BY_PATH[normalizePath(pathname)] ?? null;
}
