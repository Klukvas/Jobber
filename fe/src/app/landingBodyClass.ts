import * as React from "react";

import { getAuthModalRoute } from "@/features/auth/authModalRoutes";

/**
 * The class that mirrors the landing palette onto `<body>`.
 *
 * The landing page forces its own dark palette on a wrapper *inside* `<body>`,
 * so anything `<body>` paints outside that wrapper — the strip the cookie
 * banner reserves at the end of the document — kept the app's light
 * background and showed as a pale band across the bottom of a forced-dark
 * page. This class paints that strip in the landing colour.
 */
export const LANDING_BODY_CLASS = "landing-page";

/**
 * Whether a path is drawn with the landing presentation.
 *
 * The landing page and the three auth-modal routes over it are one page, so
 * they are one answer here too.
 */
export function isLandingPath(pathname: string): boolean {
  return pathname.replace(/\/+$/, "") === "" || !!getAuthModalRoute(pathname);
}

/**
 * Keeps the landing body class in step with the route, wherever the route came
 * from.
 *
 * The class used to be owned by the landing page component: added when it
 * mounted, removed when it unmounted. That works for a visitor who navigates,
 * and fails for everyone else. The production SPA fallback is the prerendered
 * home page, so `dist/index.html` shipped `<body class="landing-page">` and
 * *every* direct load or refresh of `/app/*`, of an auth route, of a 404 was
 * served that class. The landing page never mounted on any of them, so its
 * cleanup never ran, and the app rendered its light theme over a forced-dark
 * background — text on background at 1:1 contrast.
 *
 * So the lifecycle is owned by the route tree's root instead of by one page:
 * whatever the document arrived with, the class is present on a landing route
 * and absent on every other one, from the first commit onwards. A layout
 * effect rather than an effect, so the correction lands before the browser
 * paints the frame React mounted into.
 *
 * The prerenderer strips the class from the snapshots it writes as well — see
 * `scripts/shellHtml.mjs` — so the shipped shell does not carry it even for
 * the moment before React starts.
 */
export function useLandingBodyClass(pathname: string): void {
  React.useLayoutEffect(() => {
    const { classList } = document.body;

    if (!isLandingPath(pathname)) {
      classList.remove(LANDING_BODY_CLASS);
      return;
    }

    classList.add(LANDING_BODY_CLASS);
    return () => classList.remove(LANDING_BODY_CLASS);
  }, [pathname]);
}
