import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";

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
  reset: vi.fn(),
  set_config: vi.fn(),
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

  it("parses a decision recorded against the current policy", async () => {
    const { getStoredConsent, CONSENT_POLICY_VERSION } = await freshConsent();
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({
        choice: "accepted",
        at: "2026-01-01T00:00:00Z",
        version: CONSENT_POLICY_VERSION,
      }),
    );

    expect(getStoredConsent()).toBe("accepted");
  });

  // Consent has to be informed, and permission for a document the visitor was
  // never shown is not permission. Only an *acceptance* re-opens the banner.
  it("re-asks when an acceptance names an older policy", async () => {
    const { getStoredConsent, CONSENT_POLICY_VERSION } = await freshConsent();
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({
        choice: "accepted",
        at: "2026-01-01T00:00:00Z",
        version: CONSENT_POLICY_VERSION - 1,
      }),
    );

    expect(getStoredConsent()).toBeNull();
  });

  it("re-asks when an acceptance names no policy at all", async () => {
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "accepted", at: "2026-01-01T00:00:00Z" }),
    );
    const { getStoredConsent } = await freshConsent();

    expect(getStoredConsent()).toBeNull();
  });

  it("re-asks on a legacy plain-string acceptance", async () => {
    localStorage.setItem("cookie-consent", "accepted");
    const { getStoredConsent } = await freshConsent();

    expect(getStoredConsent()).toBeNull();
  });

  // A refusal is the other direction entirely. Nothing about a new policy
  // turns "do not track me" into a question that needs asking again, and
  // discarding it put the banner back in front of everyone who had already
  // said no — which is the nagging the refusal existed to end.
  it("keeps a refusal recorded against an older policy", async () => {
    const { getStoredConsent, CONSENT_POLICY_VERSION } = await freshConsent();
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({
        choice: "essential",
        at: "2026-01-01T00:00:00Z",
        version: CONSENT_POLICY_VERSION - 1,
      }),
    );

    expect(getStoredConsent()).toBe("essential");
  });

  it("keeps a refusal that names no policy at all", async () => {
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "essential", at: "2026-01-01T00:00:00Z" }),
    );
    const { getStoredConsent } = await freshConsent();

    expect(getStoredConsent()).toBe("essential");
  });

  it("keeps a legacy plain-string refusal", async () => {
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
    const { applyConsent, CONSENT_POLICY_VERSION } = await freshConsent();
    applyConsent("accepted");

    const stored = JSON.parse(localStorage.getItem("cookie-consent")!);
    expect(stored.choice).toBe("accepted");
    expect(stored.at).toBeTruthy();
    // Stamped with the policy it was given against, so a later change can
    // re-ask instead of assuming this answer still applies.
    expect(stored.version).toBe(CONSENT_POLICY_VERSION);

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
  /** A decision recorded against whichever policy this build ships. */
  function storeDecision(choice: string, version: number) {
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice, at: "2026-01-01T00:00:00Z", version }),
    );
  }

  it("starts analytics for a returning accepted visitor", async () => {
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    storeDecision("accepted", CONSENT_POLICY_VERSION);
    initAnalyticsIfConsented();

    expect(gtagMock.initGA4).toHaveBeenCalledOnce();
    expect(posthogLibMock.initPostHog).toHaveBeenCalledOnce();
  });

  it("does nothing without consent or with essential-only", async () => {
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    initAnalyticsIfConsented();
    storeDecision("essential", CONSENT_POLICY_VERSION);
    initAnalyticsIfConsented();

    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(posthogLibMock.initPostHog).not.toHaveBeenCalled();
  });

  // An "accepted" that has stopped counting must not leave the identity it
  // created alive while the banner asks again.
  it("purges what a superseded acceptance left behind", async () => {
    document.cookie = "_ga=GA1.1.111; path=/";
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    storeDecision("accepted", CONSENT_POLICY_VERSION - 1);

    initAnalyticsIfConsented();

    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(gtagMock.setGA4Disabled).toHaveBeenCalledWith(true);
    expect(document.cookie).not.toContain("_ga=");
    // Dropped, so the sweep happens once rather than on every page load.
    expect(localStorage.getItem("cookie-consent")).toBeNull();
  });

  // "Essential only" is an answer, and it has to survive a page load: dropping
  // it re-opened the banner on every single visit for everyone who declined.
  it("keeps an essential-only decision and starts nothing", async () => {
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    storeDecision("essential", CONSENT_POLICY_VERSION);

    initAnalyticsIfConsented();

    expect(localStorage.getItem("cookie-consent")).not.toBeNull();
    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(posthogLibMock.initPostHog).not.toHaveBeenCalled();
    // Not re-swept either: the withdrawal that produced this already ran.
    expect(gtagMock.setGA4Disabled).not.toHaveBeenCalled();
  });

  it("keeps an accepted decision it honours", async () => {
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    storeDecision("accepted", CONSENT_POLICY_VERSION);

    initAnalyticsIfConsented();

    expect(localStorage.getItem("cookie-consent")).not.toBeNull();
  });

  // The refusal outlives the policy that was refused, so a returning visitor
  // who once said no is still not tracked and is not asked again.
  it("honours a refusal from an older policy without re-sweeping", async () => {
    const { initAnalyticsIfConsented, CONSENT_POLICY_VERSION } =
      await freshConsent();
    storeDecision("essential", CONSENT_POLICY_VERSION - 1);

    initAnalyticsIfConsented();

    expect(localStorage.getItem("cookie-consent")).not.toBeNull();
    expect(gtagMock.initGA4).not.toHaveBeenCalled();
    expect(posthogLibMock.initPostHog).not.toHaveBeenCalled();
    expect(gtagMock.setGA4Disabled).not.toHaveBeenCalled();
  });

  it("leaves a first-time visitor's empty storage alone", async () => {
    const { initAnalyticsIfConsented } = await freshConsent();

    initAnalyticsIfConsented();

    expect(gtagMock.setGA4Disabled).not.toHaveBeenCalled();
  });
});

/**
 * Withdrawal stopped collection but left `_ga`, `_ga_<id>` and
 * `ph_<key>_posthog` in the jar. No new data was gathered, but the identifiers
 * survived — so granting consent again resumed the same person rather than
 * starting fresh, which is not what "essential only" promises.
 *
 * jsdom's cookie jar honours expiry, so a successful delete is observable as
 * the cookie disappearing from `document.cookie`.
 */
describe("analytics cookie cleanup on withdrawal", () => {
  function setCookies(names: string[]) {
    for (const name of names) document.cookie = `${name}=value; path=/`;
  }

  function cookieNames(): string[] {
    return document.cookie
      .split(";")
      .map((entry) => entry.split("=")[0]?.trim())
      .filter((name): name is string => !!name);
  }

  beforeEach(() => {
    for (const name of cookieNames()) {
      document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
    }
  });

  it("expires the GA client and property cookies", async () => {
    const { clearAnalyticsCookies } = await freshConsent();
    setCookies(["_ga", "_ga_S7FZHSVFJY"]);

    clearAnalyticsCookies();

    expect(cookieNames()).not.toContain("_ga");
    expect(cookieNames()).not.toContain("_ga_S7FZHSVFJY");
  });

  it("expires the PostHog store, which withdrawal used to leave behind", async () => {
    const { clearAnalyticsCookies } = await freshConsent();
    setCookies(["ph_phc_abc123_posthog"]);

    clearAnalyticsCookies();

    expect(cookieNames()).not.toContain("ph_phc_abc123_posthog");
  });

  it("expires the legacy GA session and throttle cookies", async () => {
    const { clearAnalyticsCookies } = await freshConsent();
    setCookies(["_gid", "_gat_gtag_UA_1_1"]);

    clearAnalyticsCookies();

    expect(cookieNames()).not.toContain("_gid");
    expect(cookieNames()).not.toContain("_gat_gtag_UA_1_1");
  });

  // Sign-in and consent itself must survive: this is a withdrawal of analytics,
  // not a sign-out.
  it("leaves essential cookies alone", async () => {
    const { clearAnalyticsCookies } = await freshConsent();
    setCookies(["access_token", "refresh_token", "cookie-consent", "_gargoyle"]);

    clearAnalyticsCookies();

    const remaining = cookieNames();
    expect(remaining).toContain("access_token");
    expect(remaining).toContain("refresh_token");
    expect(remaining).toContain("cookie-consent");
    // Named to look like GA without being it — the match is anchored.
    expect(remaining).toContain("_gargoyle");
  });

  it("runs as part of choosing essential-only", async () => {
    const { applyConsent } = await freshConsent();
    setCookies(["_ga", "ph_phc_abc123_posthog", "access_token"]);

    applyConsent("essential");

    expect(cookieNames()).not.toContain("_ga");
    expect(cookieNames()).not.toContain("ph_phc_abc123_posthog");
    expect(cookieNames()).toContain("access_token");
  });

  it("runs as part of reopening the banner from the footer", async () => {
    const { resetConsent } = await freshConsent();
    setCookies(["_ga"]);

    resetConsent();

    expect(cookieNames()).not.toContain("_ga");
  });

  // opt_out_capturing stops capture but keeps the distinct id; reset drops the
  // identity so re-consenting starts a new one.
  it("drops the PostHog identity as well as opting out", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    expect(posthogClientMock.opt_out_capturing).toHaveBeenCalledOnce();
    expect(posthogClientMock.reset).toHaveBeenCalledOnce();
  });

  /**
   * The order is the whole fix, not a detail. `reset()` clears PostHog's
   * persistence — including the stored opt-in/out decision — and starts a fresh
   * anonymous identity. Running it *after* `opt_out_capturing()` therefore
   * threw the opt-out away and left the client capturing again under a new id:
   * withdrawal that withdrew nothing.
   */
  it("resets the identity before opting out, never after", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    const resetOrder = posthogClientMock.reset.mock.invocationCallOrder[0];
    const optOutOrder =
      posthogClientMock.opt_out_capturing.mock.invocationCallOrder[0];

    expect(resetOrder).toBeLessThan(optOutOrder);
  });

  it("keeps that order when withdrawing from the footer too", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent, resetConsent } = await freshConsent();

    applyConsent("accepted");
    resetConsent();

    expect(
      posthogClientMock.reset.mock.invocationCallOrder[0],
    ).toBeLessThan(
      posthogClientMock.opt_out_capturing.mock.invocationCallOrder[0],
    );
  });

  it("does not touch a PostHog client that never loaded", async () => {
    posthogClientMock.__loaded = false;
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    expect(posthogClientMock.reset).not.toHaveBeenCalled();
    expect(posthogClientMock.opt_out_capturing).not.toHaveBeenCalled();
  });

  // Re-granting has to be able to start collection again.
  it("leaves analytics able to initialise after a withdrawal", async () => {
    posthogClientMock.__loaded = true;
    posthogClientMock.has_opted_out_capturing.mockReturnValue(true);
    const { applyConsent } = await freshConsent();

    applyConsent("essential");
    applyConsent("accepted");

    expect(gtagMock.setGA4Disabled).toHaveBeenLastCalledWith(false);
    expect(posthogClientMock.opt_in_capturing).toHaveBeenCalled();
  });
});

/**
 * PostHog keeps its distinct id in localStorage as well as in a cookie, and
 * `posthog.reset()` can only clear it while the library is on the page. The
 * script is loaded lazily, so the common case — accepted on an earlier visit,
 * withdrawn on this one before anything loaded it — left `ph_<key>_posthog`
 * behind and re-identified the same person on the next accept.
 */
describe("PostHog localStorage cleanup on withdrawal", () => {
  const IDENTITY_KEY = "ph_phc_abc123_posthog";
  const RATE_LIMIT_KEY = "ph_phc_abc123_posthog_rate_limit";
  const OPT_OUT_KEY = "__ph_opt_in_out_phc_abc123";

  function seedStorage() {
    localStorage.setItem(
      IDENTITY_KEY,
      JSON.stringify({ distinct_id: "person-1" }),
    );
    localStorage.setItem(RATE_LIMIT_KEY, "{}");
    localStorage.setItem(OPT_OUT_KEY, "0");
    localStorage.setItem("language", "uk");
    localStorage.setItem("resume-builder-draft", "{}");
  }

  it("removes the identity even when PostHog never loaded", async () => {
    posthogClientMock.__loaded = false;
    const { applyConsent } = await freshConsent();
    seedStorage();

    applyConsent("essential");

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
    expect(localStorage.getItem(RATE_LIMIT_KEY)).toBeNull();
  });

  it("removes the identity when PostHog is loaded", async () => {
    posthogClientMock.__loaded = true;
    const { applyConsent } = await freshConsent();
    seedStorage();

    applyConsent("essential");

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
    expect(localStorage.getItem(RATE_LIMIT_KEY)).toBeNull();
  });

  // The opt-out flag is the decision, not the identity. Deleting it would let
  // the next posthog.init() treat the visitor as undecided.
  it("keeps the opt-out flag and every unrelated key", async () => {
    const { applyConsent } = await freshConsent();
    seedStorage();

    applyConsent("essential");

    expect(localStorage.getItem(OPT_OUT_KEY)).toBe("0");
    expect(localStorage.getItem("language")).toBe("uk");
    expect(localStorage.getItem("resume-builder-draft")).toBe("{}");
  });

  it("runs from the footer's cookie-settings link too", async () => {
    const { resetConsent } = await freshConsent();
    seedStorage();

    resetConsent();

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
    expect(localStorage.getItem(OPT_OUT_KEY)).toBe("0");
  });

  // Re-consenting must be able to start collection again, from a clean slate:
  // no identity left over, and the opt-out lifted.
  it("leaves a re-consent starting from a fresh identity", async () => {
    posthogClientMock.__loaded = true;
    posthogClientMock.has_opted_out_capturing.mockReturnValue(true);
    const { applyConsent } = await freshConsent();
    seedStorage();

    applyConsent("essential");
    applyConsent("accepted");

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
    expect(posthogClientMock.opt_in_capturing).toHaveBeenCalledWith({
      captureEventName: null,
    });
    expect(gtagMock.setGA4Disabled).toHaveBeenLastCalledWith(false);
  });

  it("does not reach a key that merely looks like PostHog's", async () => {
    const { clearPostHogStorage } = await freshConsent();
    localStorage.setItem("phone_number", "x");
    localStorage.setItem("ph_posthog", "no token in the middle");
    localStorage.setItem("my_ph_abc_posthog", "not at the start");

    clearPostHogStorage();

    expect(localStorage.getItem("phone_number")).toBe("x");
    expect(localStorage.getItem("ph_posthog")).toBe("no token in the middle");
    expect(localStorage.getItem("my_ph_abc_posthog")).toBe("not at the start");
  });
});

/**
 * A cookie is only removed by a write that matches the path and domain it was
 * set with, and the browser never says which pair that was — so withdrawal has
 * to try every one it could have been. These are the two halves of that, tested
 * directly because the combined effect is invisible in jsdom's flat cookie jar.
 */
describe("cookie scope enumeration", () => {
  it("walks every path from the root down to the current one", async () => {
    const { cookiePathVariants } = await freshConsent();

    expect(cookiePathVariants("/app/jobs/job-1")).toEqual([
      "/",
      "/app",
      "/app/jobs",
      "/app/jobs/job-1",
    ]);
  });

  it("returns just the root for the root itself", async () => {
    const { cookiePathVariants } = await freshConsent();

    expect(cookiePathVariants("/")).toEqual(["/"]);
    expect(cookiePathVariants("")).toEqual(["/"]);
  });

  it("tolerates doubled and trailing separators", async () => {
    const { cookiePathVariants } = await freshConsent();

    expect(cookiePathVariants("//app//jobs/")).toEqual([
      "/",
      "/app",
      "/app/jobs",
    ]);
  });

  // The host-only form (null, no `domain=` attribute) plus each registrable
  // parent, with and without the leading dot older writers used.
  it("covers the host itself and its registrable parents", async () => {
    const { cookieDomainVariants } = await freshConsent();

    expect(cookieDomainVariants("app.example.com")).toEqual([
      null,
      "app.example.com",
      ".app.example.com",
      "example.com",
      ".example.com",
    ]);
  });

  it("never writes to a bare TLD", async () => {
    const { cookieDomainVariants } = await freshConsent();

    for (const host of ["app.example.com", "example.com", "a.b.c.example.com"]) {
      expect(cookieDomainVariants(host)).not.toContain("com");
      expect(cookieDomainVariants(host)).not.toContain(".com");
    }
  });

  it("gives a single-label host only the host-only form", async () => {
    const { cookieDomainVariants } = await freshConsent();

    expect(cookieDomainVariants("localhost")).toEqual([null]);
  });

  it("stops one level above the last label on a two-part suffix", async () => {
    const { cookieDomainVariants } = await freshConsent();

    // "co.uk" is a public suffix, so the browser rejects that write outright —
    // it is a wasted attempt, never a way to reach another site's cookies.
    expect(cookieDomainVariants("shop.co.uk")).toEqual([
      null,
      "shop.co.uk",
      ".shop.co.uk",
      "co.uk",
      ".co.uk",
    ]);
  });
});

/**
 * The regression this covers: "Accept all" then "Essential only" cleared
 * `ph_<key>_posthog` synchronously, and about 150 ms later it was back — a new
 * distinct id, and one that survived a reload. Capture stayed off, so nothing
 * was collected, but the visitor had been given a fresh identity by a
 * withdrawal whose entire point is not to keep one.
 *
 * PostHog writes its store from places the app never calls: a `/flags` request
 * that was already on the wire when consent was withdrawn lands afterwards and
 * saves the props back out. Two things answer that — persistence is switched
 * off so later saves are no-ops, and the storage is swept again over a bounded
 * window for anything that had already landed.
 */
describe("PostHog identity re-created after the withdrawal returns", () => {
  const IDENTITY_KEY = "ph_phc_abc123_posthog";
  const OPT_OUT_KEY = "__ph_opt_in_out_phc_abc123";

  /** What the SDK does on a late `/flags` response: saves its props again. */
  function simulateLateSdkWrite() {
    localStorage.setItem(
      IDENTITY_KEY,
      JSON.stringify({ distinct_id: "recreated-after-withdrawal" }),
    );
    document.cookie = `${IDENTITY_KEY}=recreated; path=/`;
  }

  beforeEach(() => {
    vi.useFakeTimers();
    posthogClientMock.__loaded = true;
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("turns PostHog's persistence off so a late save cannot land", async () => {
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    expect(posthogClientMock.set_config).toHaveBeenCalledWith({
      disable_persistence: true,
    });
  });

  // reset() and opt_out_capturing() both touch persistence themselves, so the
  // switch has to come after them or it would be flipped straight back.
  it("switches persistence off only after the identity is dropped", async () => {
    const { applyConsent } = await freshConsent();

    applyConsent("essential");

    const [optOutOrder] =
      posthogClientMock.opt_out_capturing.mock.invocationCallOrder;
    const [configOrder] = posthogClientMock.set_config.mock.invocationCallOrder;
    expect(optOutOrder).toBeLessThan(configOrder);
  });

  it("clears an identity written 150 ms after the withdrawal", async () => {
    const { applyConsent } = await freshConsent();
    applyConsent("essential");

    simulateLateSdkWrite();
    expect(localStorage.getItem(IDENTITY_KEY)).not.toBeNull();

    vi.advanceTimersByTime(200);

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
    expect(document.cookie).not.toContain(IDENTITY_KEY);
  });

  it("keeps clearing across the whole window, not just the first pass", async () => {
    const { applyConsent } = await freshConsent();
    applyConsent("essential");

    for (const at of [60, 200, 500, 1_200]) {
      simulateLateSdkWrite();
      vi.advanceTimersByTime(at);
      expect(localStorage.getItem(IDENTITY_KEY), `write before ${at}ms`).toBeNull();
    }
  });

  it("clears it from the footer's cookie-settings link too", async () => {
    const { resetConsent } = await freshConsent();
    resetConsent();

    simulateLateSdkWrite();
    vi.advanceTimersByTime(200);

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
  });

  // The withdrawal is a burst of cleanup, not a background job: a timer still
  // firing minutes later would be deleting storage nobody is writing.
  it("stops sweeping once the window has passed", async () => {
    const { applyConsent } = await freshConsent();
    applyConsent("essential");

    vi.advanceTimersByTime(10_000);
    simulateLateSdkWrite();
    vi.advanceTimersByTime(10_000);

    expect(localStorage.getItem(IDENTITY_KEY)).not.toBeNull();
    expect(vi.getTimerCount()).toBe(0);
  });

  // The opt-out is the decision and has to outlive the identity — including
  // through the sweeps, which run after it was written.
  it("leaves the opt-out flag alone throughout", async () => {
    const { applyConsent } = await freshConsent();
    localStorage.setItem(OPT_OUT_KEY, "0");

    applyConsent("essential");
    vi.advanceTimersByTime(5_000);

    expect(localStorage.getItem(OPT_OUT_KEY)).toBe("0");
  });

  it("hands persistence back when consent is granted again", async () => {
    posthogClientMock.has_opted_out_capturing.mockReturnValue(true);
    const { applyConsent } = await freshConsent();

    applyConsent("essential");
    applyConsent("accepted");

    expect(posthogClientMock.set_config).toHaveBeenLastCalledWith({
      disable_persistence: false,
    });
  });

  // Re-accepting inside the sweep window must not have the withdrawal's
  // cleanup delete the identity the new consent has just created.
  it("cancels the pending sweeps when consent is granted again", async () => {
    const { applyConsent } = await freshConsent();

    applyConsent("essential");
    vi.advanceTimersByTime(100);
    applyConsent("accepted");

    localStorage.setItem(IDENTITY_KEY, JSON.stringify({ distinct_id: "new" }));
    vi.advanceTimersByTime(5_000);

    expect(localStorage.getItem(IDENTITY_KEY)).not.toBeNull();
  });

  // The common case: accepted on an earlier visit, withdrawn on this one
  // before anything loaded the script. There is no client to configure, so the
  // sweeps are the only thing standing between the visitor and a stale id.
  it("still sweeps when PostHog never loaded", async () => {
    posthogClientMock.__loaded = false;
    const { applyConsent } = await freshConsent();

    applyConsent("essential");
    expect(posthogClientMock.set_config).not.toHaveBeenCalled();

    simulateLateSdkWrite();
    vi.advanceTimersByTime(200);

    expect(localStorage.getItem(IDENTITY_KEY)).toBeNull();
  });
});
