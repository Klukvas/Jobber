import { useEffect, useRef } from "react";
import { useLocation } from "react-router-dom";
import { cleanUrl } from "./analyticsUrl";
import { trackPageView } from "./posthog";
import { trackGA4PageView } from "./gtag";

export function usePageTracking() {
  const location = useLocation();
  const lastUrl = useRef<string | null>(null);

  useEffect(() => {
    const url = cleanUrl(location.pathname + location.search);
    // Filters and search rewrite the query as they change, which is the same
    // page once the query is cleaned.
    if (url === lastUrl.current) return;
    lastUrl.current = url;
    trackPageView(url);
    trackGA4PageView(url);
  }, [location]);
}
