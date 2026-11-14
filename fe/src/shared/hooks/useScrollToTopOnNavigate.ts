import { useEffect, useRef } from "react";
import { useLocation, useNavigationType } from "react-router-dom";

const samePathname = (pathname: string) => pathname;

/**
 * Starts each newly opened screen at its top.
 *
 * The app scrolls the window, and a client-side navigation leaves the window
 * where it was: opening Settings from the bottom of Analytics showed Settings
 * 1300px down, and a feature page opened from the landing footer showed its
 * own footer. Back and Forward (`POP`) are left alone so the browser can put
 * the user back where they were, and so is a link to a `#section`, which
 * scrolls itself.
 *
 * `screenOf` maps a pathname to the screen it shows, for routes that draw
 * over another page (the auth modals over the landing page) and must not
 * move it.
 */
export function useScrollToTopOnNavigate(
  screenOf: (pathname: string) => string = samePathname,
): void {
  const { pathname, hash } = useLocation();
  const navigationType = useNavigationType();
  const screen = screenOf(pathname);
  const previousScreen = useRef(screen);

  useEffect(() => {
    if (previousScreen.current === screen) return;
    previousScreen.current = screen;
    if (navigationType === "POP" || hash) return;
    window.scrollTo(0, 0);
  }, [screen, navigationType, hash]);
}
