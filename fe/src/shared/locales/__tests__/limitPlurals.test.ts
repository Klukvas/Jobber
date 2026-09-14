import { describe, it, expect } from "vitest";

// The real application i18n instance, not a bespoke one built for the test.
// A locally-configured instance is exactly what hid the bug this file exists
// for: the app registered Ukrainian under "ua" — a country code with no CLDR
// locale behind it — so Intl fell back to the host machine's plural rules and
// Ukrainian counts came out in whatever form the tester's browser used. The
// test only means something if it exercises the same registration the app
// ships.
import i18n from "@/shared/lib/i18n";

const COUNTED_KEYS = [
  "settings.subscription.limitReachedJobs",
  "settings.subscription.limitReachedResumes",
  "settings.subscription.limitReachedResumeBuilders",
  "settings.subscription.limitReachedAIMonthly",
  "settings.subscription.limitReachedCoverLettersUsed",
] as const;

/**
 * The counts that separate the Slavic plural categories.
 *
 * 1/21/101 are "one", 2 is "few", 5 is "many". 21 and 101 are the interesting
 * ones: they are the counts a wrong locale gets wrong quietly, because English
 * rules put them in the plural bucket where Ukrainian and Russian put them in
 * the singular.
 */
const ONE_FORM_COUNTS = [1, 21, 101];

/**
 * The subset whose noun changes shape between the categories.
 *
 * "резюме" is indeclinable — its one, few and many forms are the same string —
 * so a text comparison on it would pass no matter which form was selected.
 */
const INFLECTING_KEYS = COUNTED_KEYS.filter(
  (key) => key !== "settings.subscription.limitReachedResumes",
);

function translate(lng: string, key: string, count: number): string {
  return i18n.getFixedT(lng)(key, { count, limit: count });
}

/** The message with the number blanked out, so only the chosen form is compared. */
function form(lng: string, key: string, count: number): string {
  return translate(lng, key, count).replace(/\d+/g, "#");
}

describe("the app registers Ukrainian under a real language tag", () => {
  it("has a uk bundle and no ua one", () => {
    expect(i18n.hasResourceBundle("uk", "translation")).toBe(true);
    expect(i18n.hasResourceBundle("ua", "translation")).toBe(false);
  });

  // Without this, every assertion below would pass against English rules.
  it("resolves Ukrainian plural categories through CLDR", () => {
    const resolved = new Intl.PluralRules("uk");
    expect(resolved.select(1)).toBe("one");
    expect(resolved.select(2)).toBe("few");
    expect(resolved.select(5)).toBe("many");
    expect(resolved.select(21)).toBe("one");
    expect(resolved.select(101)).toBe("one");
  });
});

describe("plan-limit messages agree with the number", () => {
  it.each(COUNTED_KEYS)("has a distinct singular in English for %s", (key) => {
    const one = translate("en", key, 1);
    const many = translate("en", key, 10);

    expect(one).not.toBe(many);
    expect(one).toContain("1");
    expect(many).toContain("10");
  });

  it("says 'resume builder', not 'resume builders', for a limit of one", () => {
    const message = translate(
      "en",
      "settings.subscription.limitReachedResumeBuilders",
      1,
    );

    expect(message).toContain("1 resume builder.");
    expect(message).not.toContain("1 resume builders");
  });

  it("says 'resume', not 'resumes', for a limit of one", () => {
    const message = translate(
      "en",
      "settings.subscription.limitReachedResumes",
      1,
    );

    expect(message).toContain("1 resume.");
    expect(message).not.toContain("1 resumes");
  });

  it("keeps the plural for the Pro caps", () => {
    expect(
      translate("en", "settings.subscription.limitReachedResumeBuilders", 10),
    ).toContain("10 resume builders");
    expect(
      translate("en", "settings.subscription.limitReachedJobs", 100),
    ).toContain("100 tracked jobs");
  });

  // Russian and Ukrainian need three distinct forms across 1 / 2-4 / 5+.
  it.each(["ru", "uk"])(
    "resolves one, few and many separately in %s",
    (lng) => {
      const key = "settings.subscription.limitReachedResumeBuilders";

      expect(new Set([form(lng, key, 1), form(lng, key, 2), form(lng, key, 5)]).size).toBe(3);
    },
  );

  // The regression itself: 21 and 101 take the *singular* form in Ukrainian and
  // Russian. Under the invalid "ua" tag they silently took whatever the host
  // locale said, which for an English host was the plural.
  it.each(["ru", "uk"])("puts 1, 21 and 101 in the same form in %s", (lng) => {
    for (const key of COUNTED_KEYS) {
      const forms = ONE_FORM_COUNTS.map((count) => form(lng, key, count));

      expect(new Set(forms).size, `${key} in ${lng}`).toBe(1);
    }
  });

  // ...and that form is genuinely the singular, not the plural wearing the
  // same text. Only asserted for the nouns that actually inflect: "резюме" is
  // the same word in every form, so it can prove nothing either way.
  it.each(["ru", "uk"])("uses the singular, not the many form, for 21 and 101 in %s", (lng) => {
    for (const key of INFLECTING_KEYS) {
      for (const count of ONE_FORM_COUNTS) {
        expect(form(lng, key, count), `${key} at ${count} in ${lng}`).not.toBe(
          form(lng, key, 5),
        );
      }
    }
  });

  it.each(["ru", "uk"])("uses the few form for 2 and the many form for 5 in %s", (lng) => {
    const key = "settings.subscription.limitReachedJobs";

    expect(form(lng, key, 2)).not.toBe(form(lng, key, 5));
    expect(form(lng, key, 5)).toBe(form(lng, key, 100));
    expect(form(lng, key, 2)).toBe(form(lng, key, 22));
  });

  it.each(["en", "ru", "uk"])(
    "never leaves an unresolved key or a raw suffix in %s",
    (lng) => {
      for (const key of COUNTED_KEYS) {
        for (const count of [1, 2, 5, 10, 21, 100, 101]) {
          const message = translate(lng, key, count);
          expect(message).not.toContain("settings.subscription");
          expect(message).not.toMatch(/_(one|few|many|other)\b/);
          expect(message).not.toContain("{{");
        }
      }
    },
  );

  // The one limit with no number must keep working unchanged: a plan with zero
  // cover letters has nothing to count.
  it.each(["en", "ru", "uk"])("leaves the countless limit alone in %s", (lng) => {
    const message = i18n.getFixedT(lng)(
      "settings.subscription.limitReachedCoverLetters",
    );

    expect(message).not.toContain("settings.subscription");
    expect(message.length).toBeGreaterThan(10);
  });
});
