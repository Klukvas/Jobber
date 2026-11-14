import { Outlet, Navigate, useLocation } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { getAuthModalRoute } from "@/features/auth/authModalRoutes";
import { readReturnTo } from "@/shared/lib/loginReturn";

/**
 * Hosts the landing page and the three auth routes drawn on top of it.
 *
 * All four are the *same page*: `/login`, `/register` and `/forgot-password`
 * render the landing page with a modal over it. They are siblings under this
 * one layout for that reason — when the auth routes sat a level deeper, every
 * open and close moved the page component to a different depth of the route
 * tree, React unmounted and rebuilt it, and the button the visitor had just
 * pressed was replaced by a new node. Escape then returned focus to a detached
 * element, which drops it on `<body>` and strands keyboard users at the top of
 * the document. At equal depth the page is never torn down, so the opener is
 * still there to receive focus back — and the scroll position, the forced-dark
 * body class and the rest of the landing state survive the round trip too.
 *
 * The guard is per-path rather than per-layout for the same reason: somebody
 * already signed in has no business on an auth route, but the landing page
 * itself stays open to them.
 */
export function LandingLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const location = useLocation();

  // Signing in flips this guard on while the login route is still showing, so
  // it races the login form's own redirect. Both have to go to the same place
  // — the page named in `?next=` — or whichever ran last won, and the visitor
  // landed on /app/jobs instead of where they had been sent from.
  if (isAuthenticated && getAuthModalRoute(location.pathname)) {
    return <Navigate to={readReturnTo(location.search) ?? "/app"} replace />;
  }

  return <Outlet />;
}
