import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { trackPageView } from "./posthog";
import { trackGA4PageView } from "./gtag";

export function usePageTracking() {
  const location = useLocation();

  useEffect(() => {
    const url = location.pathname + location.search;
    trackPageView(url);
    trackGA4PageView(url);
  }, [location]);
}
