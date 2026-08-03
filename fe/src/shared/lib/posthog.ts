import posthog from "posthog-js";

export function initPostHog() {
  const apiKey = import.meta.env.VITE_POSTHOG_KEY;
  const apiHost = import.meta.env.VITE_POSTHOG_HOST;
  const enabled = import.meta.env.VITE_FEATURE_POSTHOG === "true";

  if (!enabled || !apiKey) return;

  posthog.init(apiKey, {
    api_host: apiHost || "https://us.i.posthog.com",
    person_profiles: "identified_only",
    capture_pageview: false, // we handle it manually via router
    capture_pageleave: true,
    autocapture: true,
  });
}

export function trackPageView(url: string) {
  if (posthog.__loaded) {
    posthog.capture("$pageview", { $current_url: url });
  }
}

export function identifyUser(userId: string, properties?: Record<string, unknown>) {
  if (posthog.__loaded) {
    posthog.identify(userId, properties);
  }
}

export function resetUser() {
  if (posthog.__loaded) {
    posthog.reset();
  }
}
