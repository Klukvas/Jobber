import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// Import translation files
import enTranslations from "@/shared/locales/en.json";
import ukTranslations from "@/shared/locales/uk.json";
import ruTranslations from "@/shared/locales/ru.json";

/**
 * Language tags, as BCP 47 spells them.
 *
 * Ukrainian is "uk". The app used to call it "ua" — that is the *country* code,
 * and there is no such locale in CLDR, so `Intl.PluralRules("ua")` fell back to
 * whatever the host machine's locale was. Ukrainian plurals then resolved with
 * English rules, and "5 резюме" came out in the singular form for anyone whose
 * browser was English. Registering the bundle under the real tag is what makes
 * one/few/many work at all.
 */
export const SUPPORTED_LANGUAGES = ["en", "uk", "ru"] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

const LANGUAGE_STORAGE_KEY = "language";

/** Tags older builds persisted, and what each one means today. */
const LEGACY_LANGUAGE_TAGS: Readonly<Record<string, SupportedLanguage>> = {
  ua: "uk",
};

/**
 * The tag to actually use for a stored or requested language.
 *
 * Anything unrecognised falls back to English rather than being passed
 * through: an unknown tag would take i18next's fallback for the *strings* while
 * still driving Intl with a locale nothing here was written for.
 */
export function normalizeLanguage(tag: string | null | undefined): SupportedLanguage {
  if (!tag) return "en";
  const migrated = LEGACY_LANGUAGE_TAGS[tag] ?? tag;
  return (SUPPORTED_LANGUAGES as readonly string[]).includes(migrated)
    ? (migrated as SupportedLanguage)
    : "en";
}

function readStoredLanguage(): SupportedLanguage {
  try {
    return normalizeLanguage(localStorage.getItem(LANGUAGE_STORAGE_KEY));
  } catch {
    // Private mode with storage denied: English until the user picks again.
    return "en";
  }
}

function persistLanguage(language: string): void {
  try {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  } catch {
    // The choice still applies to this page; it just will not be remembered.
  }
}

const resources = {
  en: {
    translation: enTranslations,
  },
  uk: {
    translation: ukTranslations,
  },
  ru: {
    translation: ruTranslations,
  },
};

const savedLanguage = readStoredLanguage();

i18n.use(initReactI18next).init({
  resources,
  lng: savedLanguage,
  fallbackLng: "en",
  interpolation: {
    escapeValue: false,
  },
});

// Rewrite the migrated tag once, so a returning user stops carrying "ua"
// around and every later read is already the real language tag.
persistLanguage(savedLanguage);

i18n.on("languageChanged", (lng) => {
  persistLanguage(lng);
  // The tags are BCP 47 already, so `lang` is the language itself — no map.
  document.documentElement.lang = lng;
});

// Set initial lang attribute
document.documentElement.lang = savedLanguage;

export default i18n;
