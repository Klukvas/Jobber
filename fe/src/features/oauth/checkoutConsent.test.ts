import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  CHECKOUT_CONSENT_KEY, CHECKOUT_CONSENT_TTL_MS,
  forgetCheckoutConsent, rememberCheckoutConsent, takeCheckoutConsent,
} from "./checkoutConsent";

const path = "/oauth/authorize?response_type=code&client_id=client&redirect_uri=https%3A%2F%2Fassistant.example%2Fcb&code_challenge=challenge&code_challenge_method=S256&state=kept&scope=mcp";
beforeEach(() => { sessionStorage.clear(); vi.useFakeTimers(); });
afterEach(() => { vi.restoreAllMocks(); sessionStorage.clear(); vi.useRealTimers(); });

describe("checkout consent destination", () => {
  it("stores only the original path/query and timestamp and consumes it once", () => {
    rememberCheckoutConsent(path);
    expect(JSON.parse(sessionStorage.getItem(CHECKOUT_CONSENT_KEY) ?? "null")).toEqual({ path, at: Date.now() });
    expect(takeCheckoutConsent()).toBe(path);
    expect(takeCheckoutConsent()).toBeNull();
  });

  it.each([
    "https://evil.example/oauth/authorize?client_id=test",
    `${window.location.origin}${path}`, `//evil.example${path}`, "/app/settings",
    `/other/..${path}`, `/oauth/authorize/../authorize${path.slice(path.indexOf("?"))}`,
    `${path}#fragment`, `${path}&code=secret`, `${path}&access_token=secret`,
    `${path}&refresh_token=secret`, `${path}&code_verifier=secret`, `${path}&unknown=anything`,
    `${path}&state=duplicate`, `${path}&state=${"a".repeat(16_384)}`,
    path.replace("S256", "plain"), path.replace("client_id=client", "client_id="),
    path.replace("/oauth/", "/oauth\\"), `${path}\n`,
  ])("refuses unsafe, unrelated or sensitive destination %s", (unsafe) => {
    rememberCheckoutConsent(path);
    rememberCheckoutConsent(unsafe);
    expect(sessionStorage.getItem(CHECKOUT_CONSENT_KEY)).toBeNull();
    expect(takeCheckoutConsent()).toBeNull();
  });

  it.each([-1, CHECKOUT_CONSENT_TTL_MS, CHECKOUT_CONSENT_TTL_MS + 1])("rejects timestamp age %i and clears it", (age) => {
    rememberCheckoutConsent(path);
    vi.setSystemTime(Date.now() + age);
    expect(takeCheckoutConsent()).toBeNull();
    expect(sessionStorage.getItem(CHECKOUT_CONSENT_KEY)).toBeNull();
  });

  it("still accepts a destination just before expiry", () => {
    rememberCheckoutConsent(path);
    vi.advanceTimersByTime(CHECKOUT_CONSENT_TTL_MS - 1);
    expect(takeCheckoutConsent()).toBe(path);
  });

  it.each([
    "bad-json", "null", "{}", JSON.stringify({ path, at: "now" }),
    JSON.stringify({ path: "https://evil.example", at: Date.now() }),
    JSON.stringify({ path: `${path}&code=secret`, at: Date.now() }),
    JSON.stringify({ path, at: Date.now(), token: "secret" }),
  ])("discards malformed or tampered storage", (raw) => {
    sessionStorage.setItem(CHECKOUT_CONSENT_KEY, raw);
    expect(takeCheckoutConsent()).toBeNull();
    expect(sessionStorage.getItem(CHECKOUT_CONSENT_KEY)).toBeNull();
  });

  it.each(["getItem", "setItem", "removeItem"] as const)("tolerates unavailable storage during %s", (method) => {
    vi.spyOn(Storage.prototype, method).mockImplementation(() => { throw new DOMException("denied", "SecurityError"); });
    expect(() => rememberCheckoutConsent(path)).not.toThrow();
    expect(takeCheckoutConsent()).toBeNull();
    expect(() => forgetCheckoutConsent()).not.toThrow();
  });
});
