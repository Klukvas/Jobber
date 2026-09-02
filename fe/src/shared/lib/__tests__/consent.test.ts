import { describe, it, expect, vi, beforeEach } from "vitest";

const gtagMock = vi.hoisted(() => ({
  initGA4: vi.fn(),
  setGA4Disabled: vi.fn(),
  trackGA4PageView: vi.fn(),
}));
const posthogLibMock = vi.hoisted(() => ({
  initPostHog: vi.fn(),
  trackPageView: vi.fn(),
}));
const posthogClientMock = vi.hoisted(() => ({
  __loaded: false,
  has_opted_out_capturing: vi.fn(() => false),
  opt_in_capturing: vi.fn(),
  opt_out_capturing: vi.fn(),
}));

vi.mock("../gtag", () => gtagMock);
vi.mock("../posthog", () => posthogLibMock);
vi.mock("posthog-js", () => ({ default: posthogClientMock }));

// The module keeps state (analyticsStarted, entryPageviewTracked) — reimport
// fresh for every test.
async function freshConsent() {
  vi.resetModules();
  return import("../consent");
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  posthogClientMock.__loaded = false;
  posthogClientMock.has_opted_out_capturing.mockReturnValue(false);
});

describe("getStoredConsent", () => {
  it("returns null when nothing is stored", async () => {
    const { getStoredConsent } = await freshConsent();
    expect(getStoredConsent()).toBeNull();
  });

  it("parses the JSON format", async () => {
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "accepted", at: "2026-01-01T00:00:00Z" }),
    );
    const { getStoredConsent } = await freshConsent();
    expect(getStoredConsent()).toBe("accepted");
  });

  it("tolerates the legacy plain-string format", async () => {
    localStorage.setItem("cookie-consent", "essential");
    const { getStoredConsent } = await freshConsent();
    expect(getStoredConsent()).toBe("essential");
  });

  it("treats corrupt values as no consent", async () => {
    const { getStoredConsent } = await freshConsent();
    for (const bad of ["{broken", '{"choice":"maybe"}', '{"at":"x"}', "42"]) {
      localStorage.setItem("cookie-consent", bad);
      expect(getStoredConsent(), bad).toBeNull();
    }
  });
});

describe("applyConsent(accepted)", () => {
  it("stores a timestamped choice, starts analytics, and tracks the entry pageview once", async () => {
    const { applyConsent } = await freshConsent();
    applyConsent("accepted");

    const stored = JSON.parse(localStorage.getItem("cookie-consent")!);
    expect(stored.choice).toBe("accepted");
    expect(stored.at).toBeTruthy();

    expect(gtagMock.initGA4).toHaveBeenCalledOnce();
    expect(posthogLibMock.initPostHog).toHaveBeenCalledOnce();
    expect(gtagMock.setGA4Disabled).toHaveBeenCalledWith(false);
    expect(gtagMock.trackGA4PageView).toHaveBeenCalledOnce();
    expect(posthogLibMock.trackPageView).toHaveBeenCalledOnce();

    // Re-accept in the same session: no double init, no duplicate entry pageview.
    applyConsent("accepted");
    expect(gtagMock.initGA4).toHaveBeenCalledOnce();
    expect(gtagMock.trackGA4PageView).toHaveBeenCalledOnce();
  });

  it("lifts PostHog's persisted opt-out so explicit consent wins", async () => {
    // Simulates a past session's opt-out surviving in PostHog's own storage.
    posthogClientMock.__loaded = true;
    posthogClientMock.has_opted_out_capturing.mockReturnValue(true);

    const { applyConsent } = await freshConsent();
    applyConsent("accepted");

    expect(posthogClientMock.opt_in_capturing).toHaveBeenCalledWith({
      captureEventName: null,
    });
  });
});

describe("applyConsent(essential) and resetConsent", () => {
  it("essential sets the GA kill switch and opts PostHog out", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    expect(gtagMock.setGA4Disabled).toHaveBeenCalledWith(true);
    expect(posthogClientMock.opt_out_capturing).toHaveBeenCalledOnce();
    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(gtagMock.trackGA4PageView).not.toHaveBeenCalled();
  });

  it("in-session withdrawal after accept stops both trackers", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent, resetConsent } = await freshConsent();

    applyConsent("accepted");
    resetConsent();

    expect(localStorage.getItem("cookie-consent")).toBeNull();
    expect(gtagMock.setGA4Disabled).toHaveBeenLastCalledWith(true);
    expect(posthogClientMock.opt_out_capturing).toHaveBeenCalled();
  });

  it("resetConsent dispatches the banner-reopen event", async () => {
    const { resetConsent, CONSENT_RESET_EVENT } = await freshConsent();
    const listener = vi.fn();
    window.addEventListener(CONSENT_RESET_EVENT, listener);

    resetConsent();

    expect(listener).toHaveBeenCalledOnce();
    window.removeEventListener(CONSENT_RESET_EVENT, listener);
  });

  it("expires _ga cookies on withdrawal", async () => {
    document.cookie = "_ga=GA1.1.111; path=/";
    document.cookie = "_ga_ABC123=GS1.1.222; path=/";
    document.cookie = "other=keep; path=/";

    const { resetConsent } = await freshConsent();
    resetConsent();

    expect(document.cookie).not.toContain("_ga=");
    expect(document.cookie).not.toContain("_ga_ABC123=");
    expect(document.cookie).toContain("other=keep");
  });
});

describe("initAnalyticsIfConsented", () => {
  it("starts analytics for a returning accepted visitor", async () => {
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "accepted", at: "2026-01-01T00:00:00Z" }),
    );
    const { initAnalyticsIfConsented } = await freshConsent();
    initAnalyticsIfConsented();

    expect(gtagMock.initGA4).toHaveBeenCalledOnce();
    expect(posthogLibMock.initPostHog).toHaveBeenCalledOnce();
  });

  it("does nothing without consent or with essential-only", async () => {
    const { initAnalyticsIfConsented } = await freshConsent();
    initAnalyticsIfConsented();
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "essential", at: "2026-01-01T00:00:00Z" }),
    );
    initAnalyticsIfConsented();

    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(posthogLibMock.initPostHog).not.toHaveBeenCalled();
  });
});
