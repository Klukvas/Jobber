import { describe, it, expect } from "vitest";

import { safeHttpsUrl } from "../safeHttpsUrl";

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
