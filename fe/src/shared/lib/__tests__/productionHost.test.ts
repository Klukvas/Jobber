import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const sdks = vi.hoisted(() => ({
  posthogInit: vi.fn(),
  sentryInit: vi.fn(),
}));

vi.mock("posthog-js", () => ({ default: { init: sdks.posthogInit } }));
vi.mock("@sentry/react", () => ({
  init: sdks.sentryInit,
  reactRouterV7BrowserTracingIntegration: vi.fn(),
  wrapCreateBrowserRouterV7: (create: unknown) => create,
}));
vi.mock("../consent", () => ({ getStoredConsent: () => null }));

import { initGA4 } from "../gtag";
import { initPostHog } from "../posthog";
import { isProductionHost, PRODUCTION_HOSTNAME } from "../productionHost";
import { initSentry } from "../sentry";

function runEveryTracker() {
  initSentry();
  initGA4();
  initPostHog();
}

function setHostname(hostname: string) {
  vi.stubGlobal("location", { ...window.location, hostname });
}

describe("the trackers' host gate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    document.head.innerHTML = "";
    vi.stubEnv("VITE_FEATURE_SENTRY", "true");
    vi.stubEnv("VITE_SENTRY_DSN", "https://key@o1.ingest.de.sentry.io/2");
    vi.stubEnv("VITE_FEATURE_GA4", "true");
    vi.stubEnv("VITE_GA4_MEASUREMENT_ID", "G-TEST");
    vi.stubEnv("VITE_FEATURE_POSTHOG", "true");
    vi.stubEnv("VITE_POSTHOG_KEY", "phc_test");
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  // The prerenderer runs a fully flagged build on localhost during the Docker
  // build, and the backend opens `http://frontend/print/resume` with a customer's
  // resume on the page. Neither may reach a third party.
  it.each(["localhost", "frontend", "staging.cursopus.com"])(
    "starts nothing on %s, even in a build with every flag on",
    (hostname) => {
      setHostname(hostname);

      runEveryTracker();

      expect(sdks.sentryInit).not.toHaveBeenCalled();
      expect(sdks.posthogInit).not.toHaveBeenCalled();
      expect(document.querySelector("script")).toBeNull();
    },
  );

  it("starts on the production host", () => {
    setHostname(PRODUCTION_HOSTNAME);

    expect(isProductionHost()).toBe(true);
    runEveryTracker();

    expect(sdks.sentryInit).toHaveBeenCalledOnce();
    expect(sdks.posthogInit).toHaveBeenCalledOnce();
    expect(document.querySelector("script")?.src).toContain("G-TEST");
  });
});
