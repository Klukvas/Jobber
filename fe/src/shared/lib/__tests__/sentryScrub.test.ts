import { describe, expect, it } from "vitest";
import { scrubBreadcrumb, scrubSentryEvent } from "../sentryScrub";

const { origin } = window.location;

describe("scrubSentryEvent", () => {
  it("cleans the page address and referrer of a report", () => {
    const event = scrubSentryEvent({
      request: {
        url: `${origin}/reset-password?email=a%40b.co&code=123456`,
        headers: { Referer: `${origin}/app/jobs?search=Acme` },
      },
    });

    expect(event.request.url).toBe(`${origin}/reset-password`);
    expect(event.request.headers.Referer).toBe(`${origin}/app/jobs`);
  });

  it("removes an email quoted in an error message", () => {
    const event = scrubSentryEvent({
      exception: {
        values: [{ type: "Error", value: "No account for jane@example.com" }],
      },
    });

    expect(JSON.stringify(event)).not.toContain("jane@example.com");
  });

  it("cleans addresses inside spans and trace data", () => {
    const event = scrubSentryEvent({
      transaction: "/s/abc123",
      spans: [{ description: "https://files.example.com/cv.pdf?sig=secret" }],
      contexts: { trace: { data: { "lcp.url": `${origin}/s/abc?x=1` } } },
    });

    expect(event.transaction).toBe("/s/:token");
    expect(JSON.stringify(event)).not.toContain("sig=secret");
    expect(event.contexts.trace.data["lcp.url"]).toBe(`${origin}/s/:token`);
  });

  it("leaves stack frames readable for the source maps to work", () => {
    const filename = `${origin}/assets/index-abc123.js`;
    const event = scrubSentryEvent({
      exception: { values: [{ stacktrace: { frames: [{ filename }] } }] },
    });

    expect(event.exception.values[0]?.stacktrace.frames[0]?.filename).toBe(
      filename,
    );
  });

  // The client keeps using these after beforeSend. A copy would have lost every
  // method, and walking one never ends.
  it("does not touch the SDK's own metadata", () => {
    const scope = { client: null as unknown };
    scope.client = scope;

    const event = scrubSentryEvent({
      sdkProcessingMetadata: { capturedSpanScope: scope },
    });

    expect(event.sdkProcessingMetadata.capturedSpanScope).toBe(scope);
  });

  it("survives a value that refers to itself", () => {
    const loop: Record<string, unknown> = {};
    loop.self = loop;

    expect(() => scrubSentryEvent({ extra: loop })).not.toThrow();
  });
});

describe("scrubBreadcrumb", () => {
  it("drops console output, which can hold anything", () => {
    expect(
      scrubBreadcrumb({ category: "console", message: "resume text" }),
    ).toBeNull();
  });

  it("cleans a navigation in both directions", () => {
    const crumb = scrubBreadcrumb({
      category: "navigation",
      data: { from: "/reset-password?email=a%40b.co", to: "/s/abc123" },
    });

    expect(crumb?.data).toEqual({ from: "/reset-password", to: "/s/:token" });
  });

  it("cleans the address of a request", () => {
    const crumb = scrubBreadcrumb({
      category: "fetch",
      data: { url: `${origin}/api/v1/jobs?search=Acme`, method: "GET" },
    });

    expect(crumb?.data?.url).toBe(`${origin}/api/v1/jobs`);
  });

  it("takes on-screen text out of a click's selector", () => {
    const crumb = scrubBreadcrumb({
      category: "ui.click",
      message: 'div > button.delete[aria-label="Delete Senior Dev at Acme"]',
    });

    expect(crumb?.message).toBe("div > button.delete");
  });
});
