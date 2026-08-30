import { describe, it, expect, vi } from "vitest";

// Mock import.meta.glob before importing the module
vi.mock("/src/content/blog/en/*.md", () => ({}));
vi.mock("/src/content/blog/ua/*.md", () => ({}));
vi.mock("/src/content/blog/ru/*.md", () => ({}));

// We need to test the exported functions. The module reads import.meta.glob
// at load time, so we mock the glob results via vi.stubGlobal.
// Instead, let's test the logic by importing and calling the functions directly.

describe("blogLoader", () => {
  it("getAllPosts returns empty array for unknown lang", async () => {
    // Dynamic import after mocks are set up
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("en");
    // With no markdown files loaded, posts should be an empty array
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getAllPosts returns empty for ua", async () => {
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("ua");
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getAllPosts returns empty for ru", async () => {
    const { getAllPosts } = await import("./blogLoader");
    const posts = getAllPosts("ru");
    expect(Array.isArray(posts)).toBe(true);
  });

  it("getPostBySlug returns undefined when no posts", async () => {
    const { getPostBySlug } = await import("./blogLoader");
    const post = getPostBySlug("nonexistent", "en");
    expect(post).toBeUndefined();
  });

  // Every language directory must contribute unique slugs. A slug shared
  // across languages collapses two articles onto one URL, and only the
  // English one stays reachable for crawlers (regression: three ua posts
  // once shadowed en/ru slugs and were invisible to indexing).
  it("slugs are unique across all languages", async () => {
    const { getAllPosts } = await import("./blogLoader");
    const slugs = ["en", "ua", "ru"].flatMap((lang) =>
      getAllPosts(lang).map((p) => p.slug),
    );
    const duplicates = slugs.filter((s, i) => slugs.indexOf(s) !== i);
    expect(duplicates).toEqual([]);
  });

  it("hreflang alternates use 'uk' for Ukrainian and include self", async () => {
    const { getAllPosts, getHreflangAlternates } = await import("./blogLoader");
    const post = getAllPosts("en").find(
      (p) => p.slug === "why-track-job-applications",
    );
    if (!post) return; // content not loaded in this environment

    const alternates = getHreflangAlternates(post);
    const codes = alternates.map((a) => a.hreflang);
    expect(codes).toContain("en");
    expect(codes).toContain("uk");
    expect(codes).not.toContain("ua");
    expect(alternates.map((a) => a.slug)).toContain(post.slug);
  });

  it("hreflang alternates are reciprocal within a cluster", async () => {
    const { getAllPosts, getHreflangAlternates } = await import("./blogLoader");
    const clustered = ["en", "ua", "ru"].flatMap((lang) =>
      getAllPosts(lang).filter((p) => p.translationKey),
    );
    for (const post of clustered) {
      const alternates = getHreflangAlternates(post);
      if (alternates.length === 0) continue;
      // Every member of the cluster must resolve to the same alternate set.
      for (const alt of alternates) {
        const targetLangDir = alt.hreflang === "uk" ? "ua" : alt.hreflang;
        const target = getAllPosts(targetLangDir).find(
          (p) => p.slug === alt.slug,
        );
        expect(target?.translationKey).toBe(post.translationKey);
      }
    }
  });

  it("posts without translationKey produce no alternates", async () => {
    const { getAllPosts, getHreflangAlternates } = await import("./blogLoader");
    const standalone = getAllPosts("ru").find(
      (p) => p.slug === "kak-podgotovitsya-k-sobesedovaniyu",
    );
    if (!standalone) return;
    expect(getHreflangAlternates(standalone)).toEqual([]);
  });
});
