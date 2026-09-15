import { describe, it, expect } from "vitest";
import { enUS, ru, uk } from "date-fns/locale";

import { formatPostDate } from "./formatPostDate";

/**
 * The old pattern was a literal `MMMM d, yyyy`: the month name came from the
 * locale, the *order* did not. Asserting on the month name alone passed either
 * way, which is how "января 15, 2025" shipped — so these assert the shape of
 * the whole sentence, day first where the language puts the day first.
 */
describe("formatPostDate", () => {
  const DATE = "2025-01-15";

  it("writes an English date the way English writes one", () => {
    expect(formatPostDate(DATE, enUS)).toBe("January 15th, 2025");
  });

  it.each([
    ["Russian", ru, "января"],
    // date-fns writes the Ukrainian day as an ordinal ("15-е"), so the day and
    // the month are checked by position rather than by an exact substring.
    ["Ukrainian", uk, "січня"],
  ] as const)("puts the day before the month in %s", (_name, locale, month) => {
    const formatted = formatPostDate(DATE, locale);

    expect(formatted).toContain(month);
    expect(formatted).toContain("2025");
    expect(formatted.indexOf("15")).toBeLessThan(formatted.indexOf(month));
  });

  it("reads the date as a plain calendar day, not a moment in this timezone", () => {
    // The front matter carries a date, not a timestamp. Parsed as UTC midnight
    // it would render as the day before anywhere west of Greenwich.
    expect(formatPostDate("2025-01-15", enUS)).toContain("15");
  });
});
