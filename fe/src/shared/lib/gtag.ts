declare global {
  interface Window {
    dataLayer: unknown[];
    gtag: (...args: unknown[]) => void;
  }
}

export function initGA4() {
  const measurementId = import.meta.env.VITE_GA4_MEASUREMENT_ID;
  const enabled = import.meta.env.VITE_FEATURE_GA4 === "true";

  if (!enabled || !measurementId) return;

  // Load gtag.js script
  const script = document.createElement("script");
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${measurementId}`;
  document.head.appendChild(script);

  // Initialize dataLayer and gtag
  window.dataLayer = window.dataLayer || [];
  window.gtag = function gtag() {
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer.push(arguments);
  };
  window.gtag("js", new Date());
  window.gtag("config", measurementId, {
    send_page_view: false, // we handle it manually via router
  });
}

export function trackGA4PageView(path: string, title?: string) {
  if (typeof window.gtag === "function") {
    window.gtag("event", "page_view", {
      page_path: path,
      page_title: title || document.title,
    });
  }
}
