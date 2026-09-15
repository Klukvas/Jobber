import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import sitemapPlugin, { writeSitemap } from "../vite-plugin-sitemap";

const POST = `---
slug: "how-to-apply"
date: "2026-01-05"
translationKey: "apply"
---

Body.
`;

describe("writeSitemap", () => {
  let root: string;
  let outDir: string;

  interface PostFields {
    readonly date?: string;
    readonly dateModified?: string;
    readonly translationKey?: string;
  }

  /** Writes one markdown post into a language directory of the blog content. */
  function writePost(
    lang: string,
    slug: string,
    { date = "2026-01-05", dateModified, translationKey }: PostFields,
  ): void {
    const dir = path.join(root, "src/content/blog", lang);
    fs.mkdirSync(dir, { recursive: true });
    const frontmatter = [
      `slug: "${slug}"`,
      `date: "${date}"`,
      ...(dateModified ? [`dateModified: "${dateModified}"`] : []),
      ...(translationKey ? [`translationKey: "${translationKey}"`] : []),
    ].join("\n");
    fs.writeFileSync(
      path.join(dir, `${slug}.md`),
      `---\n${frontmatter}\n---\n\nBody.\n`,
      "utf-8",
    );
  }

  function readSitemap(): string {
    return fs.readFileSync(path.join(outDir, "sitemap.xml"), "utf-8");
  }

  /** The `<url>…</url>` blocks, so an assertion can be scoped to one page. */
  function urlBlocks(xml: string): string[] {
    return xml.match(/<url>[\s\S]*?<\/url>/g) ?? [];
  }

  function hreflangsIn(block: string): string[] {
    return Array.from(block.matchAll(/hreflang="([^"]+)"/g)).map((m) => m[1]);
  }

  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "jobber-sitemap-"));
    outDir = path.join(root, "dist");
    vi.spyOn(console, "log").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(root, { recursive: true, force: true });
  });

  it("writes a sitemap for a root that holds the blog content", () => {
    const blogDir = path.join(root, "src/content/blog/en");
    fs.mkdirSync(blogDir, { recursive: true });
    fs.writeFileSync(path.join(blogDir, "how-to-apply.md"), POST, "utf-8");

    expect(writeSitemap(root, outDir)).toBe(true);

    const xml = fs.readFileSync(path.join(outDir, "sitemap.xml"), "utf-8");
    expect(xml).toContain(
      "<loc>https://jobber-app.com/blog/how-to-apply</loc>",
    );
  });

  it("writes nothing when the root has no blog content directory", () => {
    expect(writeSitemap(root, outDir)).toBe(false);
    expect(fs.existsSync(outDir)).toBe(false);
  });

  /**
   * The Ukrainian content lives in a directory called `ua`, which is a country
   * code and not a language tag. Google reads `hreflang` as BCP 47, so an
   * `hreflang="ua"` alternate is simply ignored — the Ukrainian article is
   * never linked to its siblings, and the cluster is published as unrelated
   * pages. The directory keeps its name; the tag it maps to is "uk".
   */
  it("publishes the ua directory under the uk language tag", () => {
    writePost("en", "how-to-apply", { translationKey: "apply" });
    writePost("ua", "yak-podavaty", { translationKey: "apply" });

    writeSitemap(root, outDir);

    const xml = readSitemap();
    expect(xml).toContain(
      '<xhtml:link rel="alternate" hreflang="uk" href="https://jobber-app.com/blog/yak-podavaty"/>',
    );
    expect(xml).not.toContain('hreflang="ua"');
  });

  it("links every member of a translation cluster to every other", () => {
    writePost("en", "how-to-apply", { translationKey: "apply" });
    writePost("ru", "kak-podavat", { translationKey: "apply" });
    writePost("ua", "yak-podavaty", { translationKey: "apply" });

    writeSitemap(root, outDir);

    const xml = readSitemap();
    for (const url of urlBlocks(xml).filter((block) =>
      block.includes("/blog/"),
    )) {
      expect(hreflangsIn(url).sort()).toEqual([
        "en",
        "ru",
        "uk",
        "x-default",
      ]);
    }
  });

  it("points x-default at the English article, not at whichever came first", () => {
    writePost("ru", "kak-podavat", { translationKey: "apply" });
    writePost("en", "how-to-apply", { translationKey: "apply" });

    writeSitemap(root, outDir);

    expect(readSitemap()).toContain(
      '<xhtml:link rel="alternate" hreflang="x-default" href="https://jobber-app.com/blog/how-to-apply"/>',
    );
  });

  it("leaves a post with no translations without alternates", () => {
    writePost("en", "solo-post", { translationKey: "solo" });

    writeSitemap(root, outDir);

    const block = urlBlocks(readSitemap()).find((b) =>
      b.includes("/blog/solo-post"),
    );
    expect(block).toBeDefined();
    expect(block).not.toContain("xhtml:link");
  });

  it("leaves a post with no translationKey out of every cluster", () => {
    writePost("en", "how-to-apply", { translationKey: "apply" });
    writePost("ru", "kak-podavat", { translationKey: "apply" });
    writePost("en", "unrelated", {});

    writeSitemap(root, outDir);

    const block = urlBlocks(readSitemap()).find((b) =>
      b.includes("/blog/unrelated"),
    );
    expect(block).not.toContain("xhtml:link");
  });

  /**
   * `lastmod` is a claim about when the page last changed, and a wrong one
   * costs re-crawls. `dateModified` is the edit date and wins over the
   * publication date whenever a post has one.
   */
  it("dates a post by its dateModified when it has been edited", () => {
    writePost("en", "how-to-apply", {
      date: "2026-01-05",
      dateModified: "2026-03-11",
    });

    writeSitemap(root, outDir);

    const block = urlBlocks(readSitemap()).find((b) =>
      b.includes("/blog/how-to-apply"),
    );
    expect(block).toContain("<lastmod>2026-03-11</lastmod>");
    expect(block).not.toContain("2026-01-05");
  });

  it("falls back to the publication date for a post never edited", () => {
    writePost("en", "how-to-apply", { date: "2026-01-05" });

    writeSitemap(root, outDir);

    const block = urlBlocks(readSitemap()).find((b) =>
      b.includes("/blog/how-to-apply"),
    );
    expect(block).toContain("<lastmod>2026-01-05</lastmod>");
  });

  it("dates the blog index by the most recently touched post", () => {
    writePost("en", "older", { date: "2026-01-05" });
    writePost("en", "newer", { date: "2026-01-06", dateModified: "2026-04-02" });

    writeSitemap(root, outDir);

    const block = urlBlocks(readSitemap()).find((b) =>
      b.includes("<loc>https://jobber-app.com/blog</loc>"),
    );
    expect(block).toContain("<lastmod>2026-04-02</lastmod>");
  });

  // A date the parser refuses is not a date, and inventing one would publish a
  // claim nobody made.
  it("ignores a malformed date rather than emitting it", () => {
    writePost("en", "how-to-apply", { date: "05/01/2026" });

    writeSitemap(root, outDir);

    const xml = readSitemap();
    expect(xml).not.toContain("05/01/2026");
    const block = urlBlocks(xml).find((b) => b.includes("/blog/how-to-apply"));
    expect(block).toMatch(/<lastmod>\d{4}-\d{2}-\d{2}<\/lastmod>/);
  });
});

describe("sitemapPlugin", () => {
  // Regression: `closeBundle` also fires in serve mode. Under Vitest that
  // resolved to the sentinel outDir "dummy-non-existing-folder", so every test
  // run created that directory next to the run's root and left a stray
  // sitemap.xml in the working tree.
  it("only applies to real builds, never to serve mode", () => {
    expect(sitemapPlugin().apply).toBe("build");
  });
});
