import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CaptureResult } from "posthog-js";

const client = vi.hoisted(() => ({
  __loaded: true,
  has_opted_out_capturing: vi.fn(() => false),
  identify: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("posthog-js", () => ({ default: client }));

async function freshPostHog() {
  vi.resetModules();
  return import("../posthog");
}

function captured(
  properties: Record<string, unknown>,
  rest: Partial<CaptureResult> = {},
): CaptureResult {
  return { uuid: "u", event: "$pageview", properties, ...rest };
}

beforeEach(() => {
  vi.clearAllMocks();
  client.__loaded = true;
  client.has_opted_out_capturing.mockReturnValue(false);
});

describe("cleanPostHogEvent", () => {
  const { origin } = window.location;
  const reset = `${origin}/reset-password?email=a%40b.co&code=123456`;

  it("cleans every address the library fills in on its own", async () => {
    const { cleanPostHogEvent } = await freshPostHog();

    const event = cleanPostHogEvent(
      captured({
        $current_url: reset,
        $pathname: "/s/abc123",
        $referrer: `${origin}/app/jobs?search=Acme`,
        $session_entry_url: reset,
        $session_entry_referrer: "https://mail.example.com/inbox/42",
        $prev_pageview_pathname: "/app/jobs/9f1c",
      }),
    );

    expect(event?.properties).toEqual({
      $current_url: `${origin}/reset-password`,
      $pathname: "/s/:token",
      $referrer: `${origin}/app/jobs`,
      $session_entry_url: `${origin}/reset-password`,
      $session_entry_referrer: "https://mail.example.com",
      $prev_pageview_pathname: "/app/jobs/:id",
    });
  });

  it("cleans what gets written onto the person", async () => {
    const { cleanPostHogEvent } = await freshPostHog();

    const event = cleanPostHogEvent(
      captured(
        {},
        {
          $set: { $current_url: reset },
          $set_once: { $initial_current_url: reset },
        },
      ),
    );

    expect(event?.$set).toEqual({ $current_url: `${origin}/reset-password` });
    expect(event?.$set_once).toEqual({
      $initial_current_url: `${origin}/reset-password`,
    });
  });

  it("keeps the marker for a direct visit", async () => {
    const { cleanPostHogEvent } = await freshPostHog();

    const event = cleanPostHogEvent(captured({ $referrer: "$direct" }));

    expect(event?.properties.$referrer).toBe("$direct");
  });

  it("does not revive an event an earlier hook dropped", async () => {
    const { cleanPostHogEvent } = await freshPostHog();

    expect(cleanPostHogEvent(null)).toBeNull();
  });
});

describe("setPostHogUser", () => {
  it("identifies by id alone", async () => {
    const { setPostHogUser } = await freshPostHog();

    setPostHogUser("user-1");

    expect(client.identify).toHaveBeenCalledExactlyOnceWith("user-1");
  });

  it("forgets the person on sign-out, so the next one does not merge in", async () => {
    const { setPostHogUser } = await freshPostHog();

    setPostHogUser("user-1");
    setPostHogUser(null);

    expect(client.reset).toHaveBeenCalledOnce();
  });

  // Runs on every page load for anybody signed out; resetting here would give a
  // returning visitor a new anonymous id each time.
  it("leaves an anonymous visitor alone", async () => {
    const { setPostHogUser } = await freshPostHog();

    setPostHogUser(null);

    expect(client.reset).not.toHaveBeenCalled();
    expect(client.identify).not.toHaveBeenCalled();
  });

  it("waits for consent when the user signed in first", async () => {
    client.__loaded = false;
    const { restorePostHogUser, setPostHogUser } = await freshPostHog();

    setPostHogUser("user-1");
    expect(client.identify).not.toHaveBeenCalled();

    client.__loaded = true;
    restorePostHogUser();
    expect(client.identify).toHaveBeenCalledExactlyOnceWith("user-1");
  });

  it("stays quiet for somebody who opted out", async () => {
    client.has_opted_out_capturing.mockReturnValue(true);
    const { setPostHogUser } = await freshPostHog();

    setPostHogUser("user-1");

    expect(client.identify).not.toHaveBeenCalled();
  });
});
