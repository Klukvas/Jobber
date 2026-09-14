import { useTranslation } from "react-i18next";
import { enUS, uk, ru } from "date-fns/locale";
import type { Locale } from "date-fns";

// Keyed by the app's BCP 47 language tags. Ukrainian is "uk"; the old "ua"
// spelling is migrated away at startup (see shared/lib/i18n).
const localeMap: Record<string, Locale> = {
  en: enUS,
  uk: uk,
  ru: ru,
};

export function useDateLocale(): Locale {
  const { i18n } = useTranslation();
  return localeMap[i18n.language] ?? enUS;
}
