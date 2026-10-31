import { cleanUrl } from "./analyticsUrl";
import { isProductionHost } from "./productionHost";

declare global {
  interface Window {
    dataLayer: unknown[];
    gtag: (...args: unknown[]) => void;
  }
}

/** Where gtag.js comes from; the only Google origin in the CSP's script-src. */
const GA_SCRIPT_ORIGIN = "https://www.googletagmanager.com";

export function initGA4() {
  const measurementId = import.meta.env.VITE_GA4_MEASUREMENT_ID;
  const enabled = import.meta.env.VITE_FEATURE_GA4 === "true";

  if (!enabled || !measurementId || !isProductionHost()) return;

  // Initialize dataLayer and gtag
  window.dataLayer = window.dataLayer || [];
  window.gtag = function gtag() {
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer.push(arguments);
  };
  window.gtag("js", new Date());
  window.gtag("config", measurementId, {
    send_page_view: false, // we handle it manually via router
    // gtag reads the address bar and the referrer on its own unless told
    // otherwise, and a reset link's address holds an email and a one-time code.
    page_location: cleanUrl(window.location.href),
    page_referrer: cleanUrl(document.referrer),
    page_title: document.title,
    allow_google_signals: false,
    allow_ad_personalization_signals: false,
    // Stores ad clicks in `_gcl_*` cookies, and no Google Ads tag reads them.
    conversion_linker: false,
  });

  // Load gtag.js script
  const script = document.createElement("script");
  script.async = true;
  script.src = `${GA_SCRIPT_ORIGIN}/gtag/js?id=${measurementId}`;
  document.head.appendChild(script);
}

/** The page as every hit reports it; see the note in `initGA4`. */
function currentPage(path: string) {
  return {
    page_location: cleanUrl(`${window.location.origin}${path}`),
    page_title: document.title,
  };
}

export function trackGA4PageView(path: string) {
  if (typeof window.gtag === "function") {
    const page = currentPage(path);
    // Points gtag's own later hits, such as user_engagement, at the same page.
    window.gtag("set", page);
    window.gtag("event", "page_view", page);
  }
}

/**
 * Sends one event and answers whether gtag took it. `onDelivered` is for an
 * event sent right before the page navigates away: it goes as a beacon, and the
 * callback fires once gtag has handed it to the browser.
 */
export function trackGA4Event(
  name: string,
  params: Readonly<Record<string, unknown>>,
  onDelivered?: () => void,
): boolean {
  if (typeof window.gtag !== "function") return false;

  window.gtag("event", name, {
    ...params,
    ...currentPage(window.location.pathname),
    ...(onDelivered
      ? { transport_type: "beacon", event_callback: onDelivered }
      : {}),
  });
  return true;
}

// Google's official kill switch: with ga-disable-<id> set, an already-loaded
// gtag.js drops every hit and stops re-setting _ga cookies. Needed for
// in-session consent withdrawal — we can't unload the script.
export function setGA4Disabled(disabled: boolean) {
  const measurementId = import.meta.env.VITE_GA4_MEASUREMENT_ID;
  if (!measurementId) return;
  (window as unknown as Record<string, unknown>)[
    `ga-disable-${measurementId}`
  ] = disabled;
}
