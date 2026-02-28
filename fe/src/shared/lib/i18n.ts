import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// Import translation files
import enTranslations from "@/shared/locales/en.json";
import uaTranslations from "@/shared/locales/ua.json";
import ruTranslations from "@/shared/locales/ru.json";

const resources = {
  en: {
    translation: enTranslations,
  },
  ua: {
    translation: uaTranslations,
  },
  ru: {
    translation: ruTranslations,
  },
};

const savedLanguage = localStorage.getItem("language") || "en";

i18n.use(initReactI18next).init({
  resources,
  lng: savedLanguage,
  fallbackLng: "en",
  interpolation: {
    escapeValue: false,
  },
});

const htmlLangMap: Record<string, string> = { ua: "uk", en: "en", ru: "ru" };

i18n.on("languageChanged", (lng) => {
  localStorage.setItem("language", lng);
  document.documentElement.lang = htmlLangMap[lng] ?? lng;
});

// Set initial lang attribute
document.documentElement.lang = htmlLangMap[savedLanguage] ?? savedLanguage;

export default i18n;
