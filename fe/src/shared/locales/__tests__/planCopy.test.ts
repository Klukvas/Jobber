import { describe, it, expect } from "vitest";

import en from "../en.json";
import ru from "../ru.json";
import uk from "../uk.json";

const LOCALES = { en, ru, uk } as const;
type Locale = keyof typeof LOCALES;
const LOCALE_NAMES = Object.keys(LOCALES) as Locale[];

/**
 * Plan caps as `be/config/plans.yaml` defines them. Duplicated as literals on
 * purpose: the frontend cannot read the backend's config at test time, and
 * writing them out is what makes a drift visible in review.
 */
const FREE_TRACKED = 25;
const PRO_TRACKED = 100;
const PRO_COVER_LETTERS = 10;

function faqFreeAnswer(locale: Locale): string {
  return LOCALES[locale].home.faq.items.free.a;
}

function pricingBullet(
  locale: Locale,
  key: "freeFeature1" | "freeFeature6" | "proFeature1" | "proFeature7",
): string {
  return LOCALES[locale].home.pricing[key];
}

/**
 * The same claims again, in the plan picker Settings opens — the one a customer
 * is looking at with a card in their hand. It drifted from the landing page in
 * both directions: "10 cover letters / month" for a cap the backend counts as a
 * lifetime total, and "1 AI match score / month" for an allowance that match
 * scoring, resume help and imports all spend from.
 */
function modalBullet(
  locale: Locale,
  key: "freeAI" | "freeJobParses" | "proCoverLetters",
): string {
  return LOCALES[locale].settings.subscription.pricing[key];
}

/**
 * The landing FAQ used to advertise "5 jobs, 5 applications" and an
 * "unlimited" Pro tier — understating the free plan five-fold and overselling
 * Pro, while the pricing card ~1500px above the same page said 25 and 100.
 */
describe("landing FAQ free-plan answer", () => {
  it("is checked against the same numbers the pricing card shows", () => {
    // If the pricing card ever moves off plans.yaml this fails first, so the
    // FAQ assertions below can be trusted to mean something.
    expect(pricingBullet("en", "freeFeature1")).toContain(String(FREE_TRACKED));
    expect(pricingBullet("en", "proFeature1")).toContain(String(PRO_TRACKED));
  });

  it.each(LOCALE_NAMES)("quotes the real free and Pro caps in %s", (locale) => {
    const answer = faqFreeAnswer(locale);

    expect(answer).toContain(String(FREE_TRACKED));
    expect(answer).toContain(String(PRO_TRACKED));
  });

  it.each(LOCALE_NAMES)(
    "does not carry the stale 5-item claim in %s",
    (locale) => {
      expect(faqFreeAnswer(locale)).not.toMatch(
        /\b5\s+(jobs|applications|вакансий|откликов|вакансій|відгуків)/i,
      );
    },
  );

  // Only Enterprise is uncapped; calling Pro "unlimited tracking" oversold a
  // plan that stops at 100.
  it.each(LOCALE_NAMES)(
    "does not describe Pro tracking as unlimited in %s",
    (locale) => {
      expect(faqFreeAnswer(locale)).not.toMatch(
        /(unlimited|безлимитн\w*|безлімітн\w*)\s+(tracking|трекинг\w*|трекінг\w*)/i,
      );
    },
  );

  // "Applications" is the only entity since the single-pipeline migration;
  // listing jobs and applications side by side implied two separate things.
  it.each(LOCALE_NAMES)(
    "does not present jobs and applications as separate entities in %s",
    (locale) => {
      expect(faqFreeAnswer(locale)).not.toMatch(
        /(jobs.*applications|вакансий.*откликов|вакансій.*відгуків)/i,
      );
    },
  );
});

/**
 * Pro's AI is unlimited in exactly two places — match scores and job parses
 * (`max_ai_requests: -1`, `max_job_parses: -1`). Cover letters are capped at
 * 10, so a bare "makes AI unlimited" promised something the plan does not do
 * the moment a customer reaches for the eleventh letter.
 */
describe("Pro's unlimited claims stay scoped to what is actually unlimited", () => {
  it.each(LOCALE_NAMES)("does not call Pro's AI unlimited outright in %s", (locale) => {
    expect(faqFreeAnswer(locale)).not.toMatch(
      /(makes AI unlimited|делает AI безлимитным|робить AI безлімітним)/i,
    );
  });

  it.each(LOCALE_NAMES)("still names the Pro cover-letter cap in %s", (locale) => {
    expect(faqFreeAnswer(locale)).toContain(String(PRO_COVER_LETTERS));
  });

  it.each(LOCALE_NAMES)(
    "keeps the pricing card's unlimited bullets on the two AI limits that are %s",
    (locale) => {
      const pricing = LOCALES[locale].home.pricing;

      // proFeature5/6 are the AI job parses and match scores: both -1.
      for (const key of ["proFeature5", "proFeature6"] as const) {
        expect(pricing[key]).toMatch(/(unlimited|безлимитн|безлімітн)/i);
      }
    },
  );
});

/**
 * Cover letters are counted with `SELECT COUNT(*) FROM cover_letters` — a
 * lifetime total, not a monthly one. The pricing card said "10 cover letters /
 * month", which is a bigger allowance than the plan has ever given.
 */
describe("cover-letter copy does not promise a monthly reset", () => {
  it.each(LOCALE_NAMES)("does not bill the Pro cover letters per month in %s", (locale) => {
    const bullet = pricingBullet(locale, "proFeature7");

    expect(bullet).toContain(String(PRO_COVER_LETTERS));
    expect(bullet).not.toMatch(/\/\s*(month|мес|міс)/i);
  });

  it.each(LOCALE_NAMES)("says the cap is a total in %s", (locale) => {
    expect(pricingBullet(locale, "proFeature7")).toMatch(
      /(in total|всего|усього)/i,
    );
  });
});

/**
 * `max_ai_requests` is one pool: match scoring, resume-builder assistance and
 * the resume autofill import all spend from it. Calling the free plan's single
 * request "1 AI match score" hid two of the three things it pays for.
 */
describe("the free AI allowance is described as the shared pool it is", () => {
  it.each(LOCALE_NAMES)("does not call the free AI allowance a match score in %s", (locale) => {
    expect(pricingBullet(locale, "freeFeature6")).not.toMatch(
      /^1 (AI match score|AI-анализ|AI-аналіз) \/ (month|мес|міс)$/i,
    );
  });

  it.each(LOCALE_NAMES)("still states one per month in %s", (locale) => {
    const bullet = pricingBullet(locale, "freeFeature6");

    expect(bullet).toContain("1");
    expect(bullet).toMatch(/(month|мес|міс)/i);
  });

  it.each(LOCALE_NAMES)("names more than one thing the request pays for in %s", (locale) => {
    const bullet = pricingBullet(locale, "freeFeature6");

    // Three comma-separated uses inside the parenthetical.
    expect(bullet).toMatch(/\(.+,.+\)/);
  });

  it.each(LOCALE_NAMES)("says the same in the FAQ in %s", (locale) => {
    expect(faqFreeAnswer(locale)).toMatch(
      /(shared|covers|общий|спільний)/i,
    );
  });
});

/**
 * The limit-reached banner is the other half of the same story: the message
 * for the AI pool must talk about the monthly allowance, and the cover-letter
 * one must not, because cover letters never reset.
 */
describe("limit-reached messages match how the backend counts", () => {
  it.each(LOCALE_NAMES)("has no plan-availability wording on the AI limit in %s", (locale) => {
    const subscription = LOCALES[locale].settings.subscription as unknown as
      Record<string, string | Record<string, string>>;

    expect(subscription.limitReachedAI).toBeUndefined();
    for (const suffix of ["_one", "_other"]) {
      expect(subscription[`limitReachedAIMonthly${suffix}`]).toBeTruthy();
    }
  });

  it.each(LOCALE_NAMES)("never promises a reset for cover letters in %s", (locale) => {
    const subscription = LOCALES[locale].settings.subscription as unknown as
      Record<string, string | Record<string, string>>;

    for (const [key, value] of Object.entries(subscription)) {
      if (!key.startsWith("limitReachedCoverLetters")) continue;
      expect(typeof value, key).toBe("string");
      expect(value as string, key).not.toMatch(/(month|мес|міс)/i);
    }
  });
});

describe("the plan picker in Settings says what the landing page says", () => {
  it.each(LOCALE_NAMES)(
    "does not bill the Pro cover letters per month in %s",
    (locale) => {
      const bullet = modalBullet(locale, "proCoverLetters");

      expect(bullet).toContain(String(PRO_COVER_LETTERS));
      expect(bullet).not.toMatch(/\/\s*(month|мес|міс)/i);
    },
  );

  it.each(LOCALE_NAMES)("calls the Pro cover-letter cap a total in %s", (locale) => {
    expect(modalBullet(locale, "proCoverLetters")).toMatch(
      /(in total|всего|усього)/i,
    );
  });

  it.each(LOCALE_NAMES)(
    "does not call the free AI allowance a match score in %s",
    (locale) => {
      expect(modalBullet(locale, "freeAI")).not.toMatch(
        /^1 (AI match score|AI-анализ|AI-аналіз) \/ (month|мес|міс)$/i,
      );
    },
  );

  it.each(LOCALE_NAMES)(
    "names more than one thing the free AI request pays for in %s",
    (locale) => {
      const bullet = modalBullet(locale, "freeAI");

      expect(bullet).toContain("1");
      expect(bullet).toMatch(/(month|мес|міс)/i);
      // Three comma-separated uses inside the parenthetical.
      expect(bullet).toMatch(/\(.+,.+\)/);
    },
  );

  // Job parses are a separate counter (`max_job_parses`), so that bullet must
  // stay per-month while the shared pool above it does not become one.
  it.each(LOCALE_NAMES)("keeps the job-parse allowance monthly in %s", (locale) => {
    expect(modalBullet(locale, "freeJobParses")).toMatch(/(month|мес|міс)/i);
  });

  it.each(LOCALE_NAMES)("matches the landing page word for word in %s", (locale) => {
    expect(modalBullet(locale, "freeAI")).toBe(
      pricingBullet(locale, "freeFeature6"),
    );
    expect(modalBullet(locale, "proCoverLetters")).toBe(
      pricingBullet(locale, "proFeature7"),
    );
  });
});
