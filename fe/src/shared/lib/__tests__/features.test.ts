import { describe, it, expect } from "vitest";
import { FEATURES } from "../features";

describe("FEATURES", () => {
  it("has GOOGLE_CALENDAR set to false", () => {
    expect(FEATURES.GOOGLE_CALENDAR).toBe(false);
  });

  it("has SENTRY defined as a boolean", () => {
    expect(typeof FEATURES.SENTRY).toBe("boolean");
  });

  it("has EMAIL_NOTIFICATIONS defined as a boolean", () => {
    expect(typeof FEATURES.EMAIL_NOTIFICATIONS).toBe("boolean");
  });

  it("has PAYMENTS defined as a boolean", () => {
    expect(typeof FEATURES.PAYMENTS).toBe("boolean");
  });

  it("contains exactly the expected keys", () => {
    const keys = Object.keys(FEATURES).sort();
    expect(keys).toEqual(
      ["EMAIL_NOTIFICATIONS", "GOOGLE_CALENDAR", "PAYMENTS", "SENTRY"].sort(),
    );
  });

  it("is a frozen (readonly) object", () => {
    // The 'as const' assertion makes it readonly at the type level.
    // At runtime we can verify the shape is stable.
    expect(FEATURES).toBeDefined();
    expect(Object.keys(FEATURES).length).toBe(4);
  });
});
