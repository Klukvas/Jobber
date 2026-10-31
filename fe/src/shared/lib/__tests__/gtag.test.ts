import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { initGA4, trackGA4Event, trackGA4PageView } from "../gtag";
import { PRODUCTION_HOSTNAME } from "../productionHost";

const RESET_LINK = "/reset-password?email=a%40b.co&code=123456";
const { origin } = window.location;

function sentArguments(): unknown[][] {
  return (window.dataLayer as IArguments[]).map((entry) => Array.from(entry));
}

beforeEach(() => {
  document.head.innerHTML = "";
  window.dataLayer = [];
  vi.stubEnv("VITE_FEATURE_GA4", "true");
  vi.stubEnv("VITE_GA4_MEASUREMENT_ID", "G-TEST");
  vi.stubGlobal("location", {
    ...window.location,
    hostname: PRODUCTION_HOSTNAME,
    href: `${origin}${RESET_LINK}`,
    pathname: "/reset-password",
    search: "?email=a%40b.co&code=123456",
  });
});

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("gtag", () => {
  it("tells gtag the cleaned page before the script reads the address bar", () => {
    initGA4();

    const config = sentArguments().find(([command]) => command === "config");
    expect(config?.[2]).toMatchObject({
      send_page_view: false,
      page_location: `${origin}/reset-password`,
      allow_google_signals: false,
      allow_ad_personalization_signals: false,
    });
  });

  it("carries the cleaned address on a page view, as page_location", () => {
    initGA4();

    trackGA4PageView(RESET_LINK);

    const view = sentArguments().find(([, name]) => name === "page_view");
    expect(view?.[2]).toMatchObject({
      page_location: `${origin}/reset-password`,
    });
    expect(JSON.stringify(sentArguments())).not.toContain("a%40b.co");
  });

  it("marks an event sent on the way out as a beacon, with a callback", () => {
    const onDelivered = vi.fn();
    initGA4();

    expect(trackGA4Event("begin_checkout", { value: 24 }, onDelivered)).toBe(
      true,
    );

    const event = sentArguments().find(([, name]) => name === "begin_checkout");
    expect(event?.[2]).toMatchObject({
      value: 24,
      transport_type: "beacon",
      event_callback: onDelivered,
    });
  });
});
