import * as Sentry from "@sentry/react";
import { useEffect } from "react";
import {
  createBrowserRouter,
  createRoutesFromChildren,
  matchRoutes,
  useLocation,
  useNavigationType,
} from "react-router-dom";
import { getStoredConsent } from "./consent";
import { isProductionHost } from "./productionHost";
import { scrubBreadcrumb, scrubSentryEvent } from "./sentryScrub";

const TRACES_SAMPLE_RATE = 0.2;

export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;
  const enabled = import.meta.env.VITE_FEATURE_SENTRY === "true";
  if (!enabled || !dsn || !isProductionHost()) return;

  Sentry.init({
    dsn,
    environment: "production",
    release: import.meta.env.VITE_SENTRY_RELEASE || undefined,
    // Error reports need no consent: they set no cookie and the privacy policy
    // describes them as sent when something breaks. A session ping on every page
    // load would not be that, and the SDK has no way to stop one on withdrawal.
    integrations: (defaults) => [
      ...defaults.filter(({ name }) => name !== "BrowserSession"),
      Sentry.reactRouterV7BrowserTracingIntegration({
        useEffect,
        useLocation,
        useNavigationType,
        createRoutesFromChildren,
        matchRoutes,
        // API timing comes from the backend's own traces; request spans here
        // would only add addresses to clean.
        traceFetch: false,
        traceXHR: false,
      }),
    ],
    // Read per trace, so a withdrawal in mid-session applies to the next one.
    tracesSampler: () =>
      getStoredConsent() === "accepted" ? TRACES_SAMPLE_RATE : 0,
    // API requests stay exactly as they are: no sentry-trace or baggage headers.
    tracePropagationTargets: [],
    beforeSend: scrubSentryEvent,
    beforeSendTransaction: scrubSentryEvent,
    beforeBreadcrumb: scrubBreadcrumb,
  });
}

export const sentryCreateBrowserRouter =
  Sentry.wrapCreateBrowserRouterV7(createBrowserRouter);
