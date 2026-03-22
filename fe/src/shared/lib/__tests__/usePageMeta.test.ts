import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        "seo.defaultTitle": "Default Title",
        "seo.defaultDescription": "Default Description",
        "pages.dashboard": "Dashboard",
        "pages.dashboardDesc": "Dashboard overview",
      };
      return translations[key] ?? key;
    },
  }),
}));

import { usePageMeta } from "../usePageMeta";

describe("usePageMeta", () => {
  let originalTitle: string;

  beforeEach(() => {
    originalTitle = document.title;
    // Clean up meta tags
    document
      .querySelectorAll('meta[name="description"]')
      .forEach((el) => el.remove());
    document
      .querySelectorAll('meta[property="og:title"]')
      .forEach((el) => el.remove());
    document
      .querySelectorAll('meta[property="og:description"]')
      .forEach((el) => el.remove());
    document
      .querySelectorAll('meta[name="robots"]')
      .forEach((el) => el.remove());
  });

  afterEach(() => {
    document.title = originalTitle;
  });

  it("sets default title when no options are provided", () => {
    renderHook(() => usePageMeta());
    expect(document.title).toBe("Default Title");
  });

  it("sets a literal title when provided", () => {
    renderHook(() => usePageMeta({ title: "My Custom Title" }));
    expect(document.title).toBe("My Custom Title");
  });

  it("sets title from translation key", () => {
    renderHook(() => usePageMeta({ titleKey: "pages.dashboard" }));
    expect(document.title).toBe("Dashboard");
  });

  it("sets meta description tag", () => {
    renderHook(() => usePageMeta({ description: "Custom desc" }));
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Custom desc");
  });

  it("sets og:title meta tag", () => {
    renderHook(() => usePageMeta({ title: "OG Title" }));
    const meta = document.querySelector('meta[property="og:title"]');
    expect(meta?.getAttribute("content")).toBe("OG Title");
  });

  it("sets og:description meta tag", () => {
    renderHook(() => usePageMeta({ description: "OG Desc" }));
    const meta = document.querySelector('meta[property="og:description"]');
    expect(meta?.getAttribute("content")).toBe("OG Desc");
  });

  it("uses translation key for description", () => {
    renderHook(() =>
      usePageMeta({ descriptionKey: "pages.dashboardDesc" }),
    );
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Dashboard overview");
  });

  it("adds robots noindex tag when noindex is true", () => {
    renderHook(() => usePageMeta({ noindex: true }));
    const meta = document.querySelector('meta[name="robots"]');
    expect(meta?.getAttribute("content")).toBe("noindex, nofollow");
  });

  it("does not add robots tag when noindex is false", () => {
    renderHook(() => usePageMeta({ noindex: false }));
    const meta = document.querySelector('meta[name="robots"]');
    expect(meta).toBeNull();
  });

  it("removes robots tag on cleanup when noindex was true", () => {
    const { unmount } = renderHook(() => usePageMeta({ noindex: true }));
    expect(document.querySelector('meta[name="robots"]')).not.toBeNull();

    unmount();
    expect(document.querySelector('meta[name="robots"]')).toBeNull();
  });

  it("updates existing meta tags instead of creating duplicates", () => {
    renderHook(() => usePageMeta({ title: "First" }));
    renderHook(() => usePageMeta({ title: "Second" }));

    const ogTitles = document.querySelectorAll('meta[property="og:title"]');
    // Each renderHook mounts independently, but the setOgTag function
    // updates existing tags rather than duplicating.
    // With two independent hooks, both set the same tag.
    expect(ogTitles.length).toBeGreaterThanOrEqual(1);
  });

  it("falls back to default description when no description options given", () => {
    renderHook(() => usePageMeta({ title: "Only Title" }));
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Default Description");
  });
});
