import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";

import { useMonthlyResetDate } from "../useMonthlyResetDate";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ i18n: { language: "en-GB" } }),
}));

afterEach(() => {
  vi.useRealTimers();
});

function resetDateAt(iso: string): string {
  vi.setSystemTime(new Date(iso));
  return renderHook(() => useMonthlyResetDate()).result.current;
}

/**
 * The monthly allowances come back when the *server's* calendar month rolls
 * over — `date_trunc('month', NOW())` in the database's own time zone, which is
 * UTC. Computing the date from the browser's local clock told anyone far enough
 * east the wrong day for a day either side of the boundary.
 */
describe("useMonthlyResetDate", () => {
  it("names the first of next month", () => {
    vi.useFakeTimers();
    expect(resetDateAt("2026-03-14T12:00:00Z")).toBe("1 April 2026");
  });

  it("rolls December over into the next year", () => {
    vi.useFakeTimers();
    expect(resetDateAt("2026-12-31T23:00:00Z")).toBe("1 January 2027");
  });

  // 12:30 on the 1st in Auckland is still 23:30 on the 31st at the server, so
  // the allowance has not come back yet — and the date to quote is the one
  // that is a few hours away, not a month.
  it("answers on the server's clock, not the visitor's", () => {
    vi.useFakeTimers();
    // 2026-03-31T23:30Z is 2026-04-01T12:30 at UTC+13.
    expect(resetDateAt("2026-03-31T23:30:00Z")).toBe("1 April 2026");
  });
});
