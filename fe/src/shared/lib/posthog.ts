import posthog, { type CaptureResult } from "posthog-js";
import { cleanUrl } from "./analyticsUrl";
import { isProductionHost } from "./productionHost";

/** EU cloud. The region of a project is fixed, and the CSP names this host. */
const POSTHOG_EU_HOST = "https://eu.i.posthog.com";

/**
 * The properties that hold an address. The library fills these from the address
 * bar on every event: `$current_url`, `$pathname`, `$referrer` and their
 * `$initial_*`, `$session_entry_*` and `$prev_pageview_*` copies.
 */
const URL_PROPERTY = /(url|pathname|referrer)$/;

/** Not an address: what `$referrer` says for a visit with no referrer. */
const DIRECT_REFERRER = "$direct";

function cleanProperties(
  properties: Readonly<Record<string, unknown>>,
): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(properties).map(([key, value]) =>
      URL_PROPERTY.test(key) &&
      typeof value === "string" &&
      value !== DIRECT_REFERRER
        ? [key, cleanUrl(value)]
        : [key, value],
    ),
  );
}

/** The last stop before an event leaves the browser. */
export function cleanPostHogEvent(
  event: CaptureResult | null,
): CaptureResult | null {
  if (!event) return null;
  return {
    ...event,
    properties: cleanProperties(event.properties),
    ...(event.$set ? { $set: cleanProperties(event.$set) } : {}),
    ...(event.$set_once ? { $set_once: cleanProperties(event.$set_once) } : {}),
  };
}

export function initPostHog() {
  const apiKey = import.meta.env.VITE_POSTHOG_KEY;
  const apiHost = import.meta.env.VITE_POSTHOG_HOST;
  const enabled = import.meta.env.VITE_FEATURE_POSTHOG === "true";

  if (!enabled || !apiKey || !isProductionHost()) return;

  posthog.init(apiKey, {
    api_host: apiHost || POSTHOG_EU_HOST,
    person_profiles: "identified_only",
    capture_pageview: false, // we handle it manually via router
    capture_pageleave: true,
    before_send: cleanPostHogEvent,
    // Resumes, contacts and job titles are on screen in this product, so nothing
    // that reads the page is on: only the events this app sends by name.
    autocapture: false,
    rageclick: false,
    capture_dead_clicks: false,
    capture_heatmaps: false,
    capture_performance: false,
    capture_exceptions: false,
    disable_session_recording: true,
    disable_surveys: true,
    disable_conversations: true,
    disable_product_tours: true,
    // With all of the above off there is nothing to fetch: no remote config, no
    // feature flags and no lazy scripts, so the CSP only needs the ingest host.
    advanced_disable_flags: true,
    disable_external_dependency_loading: true,
  });
}

export function trackPageView(url: string) {
  if (posthog.__loaded) {
    posthog.capture("$pageview", {
      $current_url: cleanUrl(`${window.location.origin}${url}`),
    });
  }
}

/** `leavingPage` skips the batch queue, which a redirect would otherwise lose. */
export function trackPostHogEvent(
  name: string,
  params: Readonly<Record<string, unknown>>,
  leavingPage = false,
) {
  if (posthog.__loaded) {
    posthog.capture(name, params, { send_instantly: leavingPage });
  }
}

/**
 * Who is signed in, kept apart from whether PostHog may hear it: consent can
 * arrive long after sign-in, and can be withdrawn and given again.
 */
let signedInUserId: string | null = null;

export function setPostHogUser(userId: string | null) {
  const signedOut = userId === null && signedInUserId !== null;
  signedInUserId = userId;
  if (signedOut && posthog.__loaded) {
    // The next person on this browser must not merge into this one.
    posthog.reset();
    return;
  }
  restorePostHogUser();
}

/** The account id and nothing else: no email, no name, no plan. */
export function restorePostHogUser() {
  if (
    signedInUserId !== null &&
    posthog.__loaded &&
    !posthog.has_opted_out_capturing()
  ) {
    posthog.identify(signedInUserId);
  }
}
