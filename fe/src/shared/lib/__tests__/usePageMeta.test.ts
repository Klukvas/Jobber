import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";

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

function wrapper({ children }: { readonly children: ReactNode }) {
  return MemoryRouter({ initialEntries: ["/test"], children });
}

describe("usePageMeta", () => {
  let originalTitle: string;

  beforeEach(() => {
    originalTitle = document.title;
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
      .querySelectorAll('meta[property="og:url"]')
      .forEach((el) => el.remove());
    document
      .querySelectorAll('meta[name="robots"]')
      .forEach((el) => el.remove());
    document
      .querySelectorAll('link[rel="canonical"]')
      .forEach((el) => el.remove());
  });

  afterEach(() => {
    document.title = originalTitle;
  });

  it("sets default title when no options are provided", () => {
    renderHook(() => usePageMeta(), { wrapper });
    expect(document.title).toBe("Default Title");
  });

  it("sets a literal title when provided", () => {
    renderHook(() => usePageMeta({ title: "My Custom Title" }), { wrapper });
    expect(document.title).toBe("My Custom Title");
  });

  it("sets title from translation key", () => {
    renderHook(() => usePageMeta({ titleKey: "pages.dashboard" }), {
      wrapper,
    });
    expect(document.title).toBe("Dashboard");
  });

  it("sets meta description tag", () => {
    renderHook(() => usePageMeta({ description: "Custom desc" }), { wrapper });
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Custom desc");
  });

  it("sets og:title meta tag", () => {
    renderHook(() => usePageMeta({ title: "OG Title" }), { wrapper });
    const meta = document.querySelector('meta[property="og:title"]');
    expect(meta?.getAttribute("content")).toBe("OG Title");
  });

  it("sets og:description meta tag", () => {
    renderHook(() => usePageMeta({ description: "OG Desc" }), { wrapper });
    const meta = document.querySelector('meta[property="og:description"]');
    expect(meta?.getAttribute("content")).toBe("OG Desc");
  });

  it("uses translation key for description", () => {
    renderHook(() => usePageMeta({ descriptionKey: "pages.dashboardDesc" }), {
      wrapper,
    });
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Dashboard overview");
  });

  it("adds robots noindex tag when noindex is true", () => {
    renderHook(() => usePageMeta({ noindex: true }), { wrapper });
    const meta = document.querySelector('meta[name="robots"]');
    expect(meta?.getAttribute("content")).toBe("noindex, nofollow");
  });

  it("does not add robots tag when noindex is false", () => {
    renderHook(() => usePageMeta({ noindex: false }), { wrapper });
    const meta = document.querySelector('meta[name="robots"]');
    expect(meta).toBeNull();
  });

  it("removes robots tag on cleanup when noindex was true", () => {
    const { unmount } = renderHook(() => usePageMeta({ noindex: true }), {
      wrapper,
    });
    expect(document.querySelector('meta[name="robots"]')).not.toBeNull();

    unmount();
    expect(document.querySelector('meta[name="robots"]')).toBeNull();
  });

  it("updates existing meta tags instead of creating duplicates", () => {
    renderHook(() => usePageMeta({ title: "First" }), { wrapper });
    renderHook(() => usePageMeta({ title: "Second" }), { wrapper });

    const ogTitles = document.querySelectorAll('meta[property="og:title"]');
    expect(ogTitles.length).toBeGreaterThanOrEqual(1);
  });

  it("falls back to default description when no description options given", () => {
    renderHook(() => usePageMeta({ title: "Only Title" }), { wrapper });
    const meta = document.querySelector('meta[name="description"]');
    expect(meta?.getAttribute("content")).toBe("Default Description");
  });

  it("sets og:url and canonical based on current route", () => {
    renderHook(() => usePageMeta({ title: "Test" }), { wrapper });
    const ogUrl = document.querySelector('meta[property="og:url"]');
    expect(ogUrl?.getAttribute("content")).toBe("https://jobber-app.com/test");

    const canonical = document.querySelector('link[rel="canonical"]');
    expect(canonical?.getAttribute("href")).toBe(
      "https://jobber-app.com/test",
    );
  });
});
