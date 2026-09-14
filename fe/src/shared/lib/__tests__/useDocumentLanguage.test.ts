import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";

import i18n from "../i18n";
import { useDocumentLanguage } from "../useDocumentLanguage";

async function setInterfaceLanguage(language: string) {
  await act(async () => {
    await i18n.changeLanguage(language);
  });
}

describe("useDocumentLanguage", () => {
  beforeEach(async () => {
    await setInterfaceLanguage("en");
  });

  afterEach(async () => {
    await setInterfaceLanguage("en");
  });

  it("declares the language of the content being shown", () => {
    renderHook(() => useDocumentLanguage("ru"));
    expect(document.documentElement.lang).toBe("ru");
  });

  it("puts the interface language back on unmount", () => {
    const { unmount } = renderHook(() => useDocumentLanguage("uk"));
    expect(document.documentElement.lang).toBe("uk");

    unmount();
    expect(document.documentElement.lang).toBe("en");
  });

  it("leaves the document alone when no language is given", () => {
    renderHook(() => useDocumentLanguage(undefined));
    expect(document.documentElement.lang).toBe("en");
  });

  it("does nothing when the content matches the interface", () => {
    const { unmount } = renderHook(() => useDocumentLanguage("en"));
    expect(document.documentElement.lang).toBe("en");

    unmount();
    expect(document.documentElement.lang).toBe("en");
  });

  it("follows a change of article without stacking overrides", () => {
    const { rerender, unmount } = renderHook(
      ({ lang }: { lang: string }) => useDocumentLanguage(lang),
      { initialProps: { lang: "ru" } },
    );
    expect(document.documentElement.lang).toBe("ru");

    rerender({ lang: "uk" });
    expect(document.documentElement.lang).toBe("uk");

    unmount();
    expect(document.documentElement.lang).toBe("en");
  });

  /**
   * i18n writes `<html lang>` itself on every language change, so the two
   * writers were racing: the override won at mount, i18n won on the next
   * switch, and the article was left declared in the wrong language.
   */
  describe("while the override is up, the interface language does not steal it", () => {
    it("keeps the article's language when the interface switches", async () => {
      renderHook(() => useDocumentLanguage("ru"));

      await setInterfaceLanguage("uk");

      expect(document.documentElement.lang).toBe("ru");
    });

    it("survives several switches", async () => {
      renderHook(() => useDocumentLanguage("ru"));

      await setInterfaceLanguage("uk");
      await setInterfaceLanguage("en");
      await setInterfaceLanguage("uk");

      expect(document.documentElement.lang).toBe("ru");
    });

    // The other half of the race: unmounting restored whatever was on the
    // element at mount, so leaving the article put the *pre-switch* language
    // back on the whole app.
    it("restores the language the interface is in now, not the one it was in", async () => {
      const { unmount } = renderHook(() => useDocumentLanguage("ru"));
      await setInterfaceLanguage("uk");

      unmount();

      expect(document.documentElement.lang).toBe("uk");
      expect(document.documentElement.lang).toBe(i18n.language);
    });

    // Ukrainian is "uk" everywhere now: the interface tag, the document
    // attribute and the blog's own hreflang all agree.
    it("uses the BCP 47 tag for Ukrainian, never the country code", async () => {
      await setInterfaceLanguage("uk");
      const { unmount } = renderHook(() => useDocumentLanguage("ru"));

      unmount();

      expect(document.documentElement.lang).toBe("uk");
    });

    it("stops listening once it is gone", async () => {
      const { unmount } = renderHook(() => useDocumentLanguage("ru"));
      unmount();

      await setInterfaceLanguage("uk");

      expect(document.documentElement.lang).toBe("uk");
    });
  });
});
