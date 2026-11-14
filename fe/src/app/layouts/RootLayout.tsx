import { Outlet, useLocation } from "react-router-dom";
import { Suspense } from "react";
import { usePageTracking } from "@/shared/lib/usePageTracking";
import { useLandingBodyClass } from "@/app/landingBodyClass";
import { getAuthModalRoute } from "@/features/auth/authModalRoutes";
import { useScrollToTopOnNavigate } from "@/shared/hooks/useScrollToTopOnNavigate";

// An auth modal is drawn over the landing page: opening or closing it is the
// same screen and must not scroll the page behind it.
const screenOf = (pathname: string) =>
  getAuthModalRoute(pathname) ? "/" : pathname;

export function RootLayout() {
  const { pathname } = useLocation();

  usePageTracking();
  // Owned here rather than by the landing page: the production SPA fallback is
  // the prerendered home page, so a direct load of any other route arrives
  // with the landing class already on <body> and no landing page to take it
  // off again.
  useLandingBodyClass(pathname);
  // Here, not in AppLayout, so the public pages (features, blog, legal) open
  // at their top as well.
  useScrollToTopOnNavigate(screenOf);

  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      }
    >
      <Outlet />
    </Suspense>
  );
}
