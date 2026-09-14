import { describe, it, expect } from "vitest";

import {
  getAllPosts,
  getHreflangAlternates,
  getPostBySlug,
  loadPosts,
  postLanguageTag,
} from "./blogLoader";

/**
 * The real content, not a stub.
 *
 * `import.meta.glob` is resolved by Vite at build time, so the module under
 * test is loaded here with every markdown file in `src/content/blog` already
 * inlined — the same posts the site ships. The previous version of this file
 * tried to `vi.mock` the glob patterns, which resolve to nothing, and then
 * guarded every assertion with `if (!post) return`: the suite passed whether
 * the content loaded or not, and would have gone on passing if the loader had
 * returned nothing at all.
 */
const LANGUAGES = ["en", "ua", "ru"] as const;

describe("blogLoader — the shipped content", () => {
  it("loads every markdown file in each language directory", () => {
    // These are counts of files on disk. A drop means the glob stopped
    // matching; a mismatch after adding an article is this test asking to be
    // updated, which is the point.
    expect(getAllPosts("en")).toHaveLength(8);
    expect(getAllPosts("ua")).toHaveLength(13);
    expect(getAllPosts("ru")).toHaveLength(6);
  });

  it("treats 'uk' and 'ua' as the same content directory", () => {
    expect(getAllPosts("uk")).toEqual(getAllPosts("ua"));
  });

  it("falls back to English for a language with no content", () => {
    expect(getAllPosts("de")).toEqual(getAllPosts("en"));
  });

  it("gives every post the metadata its markup depends on", () => {
    for (const lang of LANGUAGES) {
      for (const post of getAllPosts(lang)) {
        expect(post.slug).toMatch(/^[a-z0-9-]+$/);
        expect(post.title.length).toBeGreaterThan(0);
        expect(post.description.length).toBeGreaterThan(0);
        // Parsed by `parseISO` for sorting and printed into `datePublished`.
        expect(post.date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
        expect(post.tags.length).toBeGreaterThan(0);
        expect(post.content.length).toBeGreaterThan(0);
        expect(post.lang).toBe(lang === "en" ? "en" : lang);
      }
    }
  });

  it("orders each language newest first", () => {
    for (const lang of LANGUAGES) {
      const dates = getAllPosts(lang).map((post) => post.date);
      expect(dates).toEqual([...dates].sort().reverse());
    }
  });

  // A slug shared across languages collapses two articles onto one URL, and
  // only the English one stays reachable for crawlers (regression: three ua
  // posts once shadowed en/ru slugs and were invisible to indexing).
  it("keeps slugs unique across all languages", () => {
    const slugs = LANGUAGES.flatMap((lang) =>
      getAllPosts(lang).map((post) => post.slug),
    );

    expect(new Set(slugs).size).toBe(slugs.length);
  });
});

describe("blogLoader — looking a post up", () => {
  it("resolves a slug in the requested language", () => {
    const post = getPostBySlug("why-track-job-applications", "en");

    expect(post?.lang).toBe("en");
    expect(post?.title).toBe(
      "Why You Should Track Every Job Application (And How to Do It)",
    );
  });

  // A shared link or a crawler hits a localized URL under whatever interface
  // language the visitor happens to have.
  it("still resolves a localized slug under a different interface language", () => {
    const post = getPostBySlug("kak-podgotovitsya-k-sobesedovaniyu", "en");

    expect(post?.lang).toBe("ru");
  });

  it("returns nothing for a slug no language has", () => {
    expect(getPostBySlug("no-such-article", "en")).toBeUndefined();
  });
});

describe("blogLoader — language annotations", () => {
  it("tags the ua directory as 'uk' and never as 'ua'", () => {
    for (const post of getAllPosts("ua")) {
      expect(postLanguageTag(post)).toBe("uk");
    }
  });

  it("tags English and Russian posts with their own codes", () => {
    expect(postLanguageTag(getAllPosts("en")[0])).toBe("en");
    expect(postLanguageTag(getAllPosts("ru")[0])).toBe("ru");
  });

  it("emits the whole cluster for a translated post", () => {
    const post = getPostBySlug("why-track-job-applications", "en");
    if (!post) throw new Error("the English tracking article is missing");

    const alternates = getHreflangAlternates(post);

    expect(alternates.map((alternate) => alternate.hreflang)).toEqual([
      "en",
      "uk",
      "ru",
    ]);
    expect(alternates.map((alternate) => alternate.slug)).toContain(post.slug);
  });

  // Without this an RU-only article opened under an English UI emitted no
  // language signal at all.
  it("emits a self-referencing alternate for a post with no translations", () => {
    const post = getPostBySlug("kak-podgotovitsya-k-sobesedovaniyu", "ru");
    if (!post) throw new Error("the Russian interview article is missing");

    expect(getHreflangAlternates(post)).toEqual([
      { hreflang: "ru", slug: post.slug },
    ]);
  });

  it("keeps every cluster reciprocal", () => {
    const clustered = LANGUAGES.flatMap((lang) =>
      getAllPosts(lang).filter((post) => post.translationKey),
    );
    expect(clustered.length).toBeGreaterThan(0);

    for (const post of clustered) {
      const alternates = getHreflangAlternates(post);
      expect(alternates.length).toBeGreaterThan(1);

      for (const alternate of alternates) {
        const directory = alternate.hreflang === "uk" ? "ua" : alternate.hreflang;
        const target = getAllPosts(directory).find(
          (candidate) => candidate.slug === alternate.slug,
        );
        expect(target?.translationKey).toBe(post.translationKey);
      }
    }
  });
});

describe("blogLoader — frontmatter mapping", () => {
  const markdown = (frontmatter: string, body = "Body text.") =>
    `---\n${frontmatter}\n---\n${body}\n`;

  it("carries every declared field onto the post", () => {
    const [post] = loadPosts({
      "/src/content/blog/en/a.md": markdown(
        [
          'title: "Ten resume tips"',
          'slug: "ten-resume-tips"',
          'date: "2026-03-01"',
          'dateModified: "2026-04-02"',
          'description: "What to cut first."',
          'tags: ["resume", "ats"]',
          'lang: "en"',
          'image: "/blog/resume.png"',
          'translationKey: "resume-tips"',
        ].join("\n"),
        "# Ten resume tips",
      ),
    });

    expect(post).toEqual({
      title: "Ten resume tips",
      slug: "ten-resume-tips",
      date: "2026-03-01",
      // Both used by the article's JSON-LD, and both were dropped on the
      // floor before: a post that declared them advertised the publication
      // date and the site-wide OG card instead.
      dateModified: "2026-04-02",
      description: "What to cut first.",
      tags: ["resume", "ats"],
      lang: "en",
      image: "/blog/resume.png",
      content: "# Ten resume tips",
      translationKey: "resume-tips",
    });
  });

  it("leaves optional fields undefined rather than inventing them", () => {
    const [post] = loadPosts({
      "/src/content/blog/en/a.md": markdown(
        ['title: "Plain"', 'slug: "plain"', 'date: "2026-03-01"'].join("\n"),
      ),
    });

    expect(post.dateModified).toBeUndefined();
    expect(post.image).toBeUndefined();
    expect(post.translationKey).toBeUndefined();
    expect(post.tags).toEqual([]);
    expect(post.lang).toBe("en");
  });

  it("sorts newest first regardless of file order", () => {
    const posts = loadPosts({
      "/src/content/blog/en/old.md": markdown(
        ['title: "Old"', 'slug: "old"', 'date: "2025-01-01"'].join("\n"),
      ),
      "/src/content/blog/en/new.md": markdown(
        ['title: "New"', 'slug: "new"', 'date: "2026-06-01"'].join("\n"),
      ),
    });

    expect(posts.map((post) => post.slug)).toEqual(["new", "old"]);
  });

  it("keeps a file with no frontmatter as pure content", () => {
    const [post] = loadPosts({ "/src/content/blog/en/a.md": "Just prose." });

    expect(post.content).toBe("Just prose.");
    expect(post.slug).toBe("");
  });
});
