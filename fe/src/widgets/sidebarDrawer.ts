import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { useSidebarStore } from "@/stores/sidebarStore";

/** Below this width the sidebar is a drawer that slides over the page. */
export const MOBILE_VIEWPORT = "(max-width: 767px)";

export const isSidebarDrawerOpen = (
  isSlideOver: boolean,
  isMobileOpen: boolean,
): boolean => isSlideOver && isMobileOpen;

/**
 * True while the phone drawer covers the page. The layout makes the page
 * behind it `inert`, so a screen reader cannot wander out of the drawer either.
 */
export function useIsSidebarDrawerOpen(): boolean {
  const isMobileOpen = useSidebarStore((state) => state.isMobileOpen);
  const isSlideOver = useMediaQuery(MOBILE_VIEWPORT);
  return isSidebarDrawerOpen(isSlideOver, isMobileOpen);
}
