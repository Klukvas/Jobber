import { useEffect } from "react";
// The i18next singleton itself, not the app's wrapper around it: the wrapper
// pulls in react-i18next, and this hook only needs the language and its change
// event. `@/shared/lib/i18n` configures this very instance.
import i18n from "i18next";

/**
 * Overrides `<html lang>` for a page whose content is in a different language
 * from the interface, and hands the attribute back on unmount.
 *
 * A Russian article opened with an English UI used to be served under
 * `lang="en"`, which tells screen readers to pronounce it with English rules
 * and tells search engines the wrong thing about the page.
 *
 * Two writers share the attribute, and that is what makes this more than a
 * one-line effect: i18n sets `lang` on every `languageChanged`. Without the
 * listener below, switching the interface language while an article was open
 * silently took the page back to the interface language even though a Russian
 * article was still on screen. And restoring a *snapshot* taken at mount put
 * the pre-switch language back on the way out, so the whole app was left
 * mislabelled — the current interface language is the only correct thing to
 * restore.
 *
 * Pass an empty/undefined language to leave the document alone.
 */
export function useDocumentLanguage(lang: string | undefined) {
  useEffect(() => {
    if (!lang) return;

    const root = document.documentElement;
    const applyOverride = () => {
      if (root.lang !== lang) root.lang = lang;
    };
    applyOverride();

    i18n.on("languageChanged", applyOverride);
    return () => {
      i18n.off("languageChanged", applyOverride);
      // `fallbackLng` if nothing has been initialised yet, which only happens
      // outside the running app.
      root.lang = i18n.language || "en";
    };
  }, [lang]);
}
