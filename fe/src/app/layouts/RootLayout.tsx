import { Outlet, useLocation } from "react-router-dom";
import { Suspense } from "react";
import { usePageTracking } from "@/shared/lib/usePageTracking";
import { useLandingBodyClass } from "@/app/landingBodyClass";

export function RootLayout() {
  const { pathname } = useLocation();

  usePageTracking();
  // Owned here rather than by the landing page: the production SPA fallback is
  // the prerendered home page, so a direct load of any other route arrives
  // with the landing class already on <body> and no landing page to take it
  // off again.
  useLandingBodyClass(pathname);

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
