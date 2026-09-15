import { describe, it, expect, beforeEach, vi } from "vitest";

import {
  PRERENDER_CONSENT_KEY,
  PRERENDER_CONSENT_RECORD,
} from "../prerenderConsent.mjs";

vi.mock("../../src/shared/lib/gtag", () => ({
  initGA4: vi.fn(),
  setGA4Disabled: vi.fn(),
  trackGA4PageView: vi.fn(),
}));
vi.mock("../../src/shared/lib/posthog", () => ({
  initPostHog: vi.fn(),
  trackPageView: vi.fn(),
}));
vi.mock("posthog-js", () => ({
  default: {
    __loaded: false,
    has_opted_out_capturing: vi.fn(() => false),
    opt_in_capturing: vi.fn(),
    opt_out_capturing: vi.fn(),
    reset: vi.fn(),
    set_config: vi.fn(),
  },
}));

beforeEach(() => {
  localStorage.clear();
  vi.resetModules();
});

/**
 * The seeded record used to carry a hand-copied `version: 1`. Nothing tied it
 * to the app's CONSENT_POLICY_VERSION, so bumping that constant would have
 * silently made the seed unhonourable — and every prerendered page would ship
 * with the cookie banner in it. This is that guard, run against the app's own
 * reader rather than against a second copy of its rules.
 */
describe("the consent record the prerenderer seeds", () => {
  it("is a decision the app honours, so no snapshot is rendered with the banner", async () => {
    const { getStoredConsent } = await import("../../src/shared/lib/consent");
    localStorage.setItem(
      PRERENDER_CONSENT_KEY,
      JSON.stringify(PRERENDER_CONSENT_RECORD),
    );

    expect(getStoredConsent()).toBe("essential");
  });

  it("survives a policy-version bump, because it names no version", async () => {
    const { getStoredConsent, CONSENT_POLICY_VERSION } = await import(
      "../../src/shared/lib/consent"
    );

    expect(PRERENDER_CONSENT_RECORD).not.toHaveProperty("version");
    // Named only to make the coupling explicit: whatever this becomes, the
    // seeded refusal above is still honoured.
    expect(typeof CONSENT_POLICY_VERSION).toBe("number");
    localStorage.setItem(
      PRERENDER_CONSENT_KEY,
      JSON.stringify(PRERENDER_CONSENT_RECORD),
    );

    expect(getStoredConsent()).toBe("essential");
  });

  it("starts no analytics during a build", async () => {
    const { initAnalyticsIfConsented } = await import(
      "../../src/shared/lib/consent"
    );
    const { initGA4 } = await import("../../src/shared/lib/gtag");
    const { initPostHog } = await import("../../src/shared/lib/posthog");
    localStorage.setItem(
      PRERENDER_CONSENT_KEY,
      JSON.stringify(PRERENDER_CONSENT_RECORD),
    );

    initAnalyticsIfConsented();

    expect(initGA4).not.toHaveBeenCalled();
    expect(initPostHog).not.toHaveBeenCalled();
    // And the record is still there: nothing swept it as unreadable.
    expect(localStorage.getItem(PRERENDER_CONSENT_KEY)).not.toBeNull();
  });
});
