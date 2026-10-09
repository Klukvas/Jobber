import { describe, it, expect } from "vitest";

import {
  BILLING_HOST_SUFFIX,
  safeBillingUrl,
  safeHttpsUrl,
} from "../safeHttpsUrl";

describe("safeHttpsUrl", () => {
  it("accepts an absolute https URL and returns it normalized", () => {
    expect(safeHttpsUrl("https://www.creem.io/checkout/1?a=b#c")).toBe(
      "https://www.creem.io/checkout/1?a=b#c",
    );
  });

  it.each([
    ["http", "http://www.creem.io/x"],
    ["javascript:", "javascript:alert(1)"],
    ["data:", "data:text/html,<script>1</script>"],
    ["relative", "/checkout/1"],
    ["protocol-relative", "//evil.example/checkout"],
    ["no host", "https://"],
    ["username only", "https://creem.io@evil.com/x"],
    ["username and password", "https://user:pw@creem.io/x"],
    ["password only", "https://:pw@creem.io/x"],
    ["blank", ""],
  ])("rejects %s", (_label, raw) => {
    expect(safeHttpsUrl(raw)).toBeNull();
  });

  it.each([[42], [null], [undefined], [{}]])(
    "rejects the non-string %j",
    (raw) => {
      expect(safeHttpsUrl(raw)).toBeNull();
    },
  );
});

describe("safeBillingUrl", () => {
  it.each([
    [
      "a checkout URL, returned unchanged",
      "https://www.creem.io/test/checkout/ch_1?x=1#frag",
    ],
    ["the bare domain", "https://creem.io/my-orders/login/abc"],
    ["an upper-case host", "https://CHECKOUT.CREEM.IO/x"],
  ])("accepts %s", (_label, raw) => {
    const expected = new URL(raw).href;
    expect(safeBillingUrl(raw)).toBe(expected);
  });

  it("returns a clean URL byte for byte", () => {
    const raw = "https://www.creem.io/test/checkout/ch_1?x=1#frag";
    expect(safeBillingUrl(raw)).toBe(raw);
  });

  it.each([
    ["javascript:", "javascript:alert(1)"],
    ["upper-case javascript:", "JAVASCRIPT:alert(1)"],
    ["leading space", " https://www.creem.io/x"],
    ["leading newline", "\nhttps://www.creem.io/x"],
    ["embedded tab", "ht\ttps://www.creem.io/x"],
    ["backslashes", "https:\\\\www.creem.io\\x"],
    ["a single slash", "https:/www.creem.io/x"],
    ["a backslash before @", "https://evil.test\\@creem.io"],
    ["userinfo naming creem", "https://creem.io@evil.test"],
    ["userinfo with password", "https://creem.io:pass@evil.test"],
    ["a lookalike suffix", "https://evilcreem.io/x"],
    ["creem.io as a subdomain of another host", "https://creem.io.evil.test/x"],
    ["a hyphenated lookalike", "https://creem.io-evil.test/x"],
    ["a non-default port", "https://www.creem.io:8443/x"],
    ["http", "http://www.creem.io/x"],
    ["protocol-relative", "//www.creem.io/x"],
    ["over-long input", "https://www.creem.io/" + "a".repeat(3000)],
    ["data:", "data:text/html,x"],
    ["another https host", "https://evil.test/checkout"],
  ])("rejects %s", (_label, raw) => {
    expect(safeBillingUrl(raw)).toBeNull();
  });

  it.each([[42], [null], [undefined], [{}], [["https://www.creem.io/x"]]])(
    "rejects the non-string %j",
    (raw) => {
      expect(safeBillingUrl(raw)).toBeNull();
    },
  );

  it("accepts a URL exactly at the length limit", () => {
    const prefix = `https://www.${BILLING_HOST_SUFFIX}/`;
    const raw = prefix + "a".repeat(2048 - prefix.length);
    expect(raw).toHaveLength(2048);
    expect(safeBillingUrl(raw)).toBe(raw);
  });
});
