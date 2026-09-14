import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const LANGUAGE_KEY = "language";

type I18nModule = typeof import("../i18n");

async function loadI18n(): Promise<I18nModule> {
  vi.resetModules();
  return import("../i18n");
}

/**
 * Ukrainian used to be persisted as "ua". That is a country code — CLDR has no
 * such locale — so `Intl.PluralRules` fell back to the host machine's rules and
 * Ukrainian counts resolved with whatever forms the browser's own language
 * used. The tag is "uk", and anyone who chose Ukrainian before the change has
 * to land on it without being silently reset to English.
 */
describe("language tag migration", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.lang = "";
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("keeps a returning Ukrainian user on Ukrainian", async () => {
    localStorage.setItem(LANGUAGE_KEY, "ua");

    const { default: i18n } = await loadI18n();

    expect(i18n.language).toBe("uk");
  });

  it("rewrites the stored tag once, so the old one stops being carried around", async () => {
    localStorage.setItem(LANGUAGE_KEY, "ua");

    await loadI18n();

    expect(localStorage.getItem(LANGUAGE_KEY)).toBe("uk");
  });

  it("publishes the BCP 47 tag as the document language", async () => {
    localStorage.setItem(LANGUAGE_KEY, "ua");

    await loadI18n();

    expect(document.documentElement.lang).toBe("uk");
  });

  it("keeps the document language in step with a later switch", async () => {
    const { default: i18n } = await loadI18n();

    await i18n.changeLanguage("uk");

    expect(document.documentElement.lang).toBe("uk");
    expect(localStorage.getItem(LANGUAGE_KEY)).toBe("uk");
  });

  it("leaves the other languages exactly as they were", async () => {
    localStorage.setItem(LANGUAGE_KEY, "ru");

    const { default: i18n } = await loadI18n();

    expect(i18n.language).toBe("ru");
    expect(localStorage.getItem(LANGUAGE_KEY)).toBe("ru");
  });

  it("falls back to English for nothing stored or a tag we do not ship", async () => {
    const { normalizeLanguage } = await loadI18n();

    expect(normalizeLanguage(null)).toBe("en");
    expect(normalizeLanguage("")).toBe("en");
    expect(normalizeLanguage("fr")).toBe("en");
    expect(normalizeLanguage("ua")).toBe("uk");
    expect(normalizeLanguage("uk")).toBe("uk");
    expect(normalizeLanguage("ru")).toBe("ru");
  });

  // A denied storage jar must not take the app down on the very first import.
  it("still starts when localStorage throws", async () => {
    const getItem = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new Error("access denied");
      });
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("access denied");
      });

    try {
      const { default: i18n } = await loadI18n();
      expect(i18n.language).toBe("en");
    } finally {
      getItem.mockRestore();
      setItem.mockRestore();
    }
  });
});
