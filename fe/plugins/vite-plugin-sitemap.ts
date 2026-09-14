import fs from "node:fs";
import path from "node:path";
import type { Plugin, ResolvedConfig } from "vite";

const VALID_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const VALID_DATE = /^\d{4}-\d{2}-\d{2}$/;

interface SitemapAlternate {
  readonly hreflang: string;
  readonly href: string;
}

interface SitemapEntry {
  readonly loc: string;
  readonly changefreq: string;
  readonly priority: string;
  readonly lastmod?: string;
  readonly alternates?: readonly SitemapAlternate[];
}

interface BlogEntry {
  readonly slug: string;
  readonly lastmod: string;
  readonly translationKey: string | null;
  // hreflang code derived from the content directory (ua dir → "uk").
  readonly hreflang: string;
}

const HREFLANG_BY_DIR: Record<string, string> = {
  en: "en",
  ua: "uk",
  ru: "ru",
};

function escapeXml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

// Mirrors the frontmatter parser in blogLoader.ts — keep in sync.
function extractFromFrontmatter(content: string): {
  slug: string | null;
  date: string | null;
  dateModified: string | null;
  translationKey: string | null;
} {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match)
    return { slug: null, date: null, dateModified: null, translationKey: null };

  let slug: string | null = null;
  let date: string | null = null;
  let dateModified: string | null = null;
  let translationKey: string | null = null;

  for (const line of match[1].split("\n")) {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) continue;
    const key = line.slice(0, colonIndex).trim();
    const raw = line
      .slice(colonIndex + 1)
      .trim()
      .replace(/^"|"$/g, "");

    if (key === "slug" && VALID_SLUG.test(raw)) slug = raw;
    else if (key === "date" && VALID_DATE.test(raw)) date = raw;
    else if (key === "dateModified" && VALID_DATE.test(raw)) dateModified = raw;
    else if (key === "translationKey" && VALID_SLUG.test(raw))
      translationKey = raw;
  }

  return { slug, date, dateModified, translationKey };
}

// Localized posts have unique per-language slugs, so every language directory
// contributes its own /blog/<slug> URLs. Dedup by slug in case one is shared
// across languages (it maps to the same URL).
function collectBlogEntries(blogDir: string): BlogEntry[] {
  const entries: BlogEntry[] = [];
  const seen = new Set<string>();

  for (const lang of ["en", "ru", "ua"]) {
    const dir = path.join(blogDir, lang);
    if (!fs.existsSync(dir)) continue;
    for (const file of fs.readdirSync(dir)) {
      if (!file.endsWith(".md")) continue;
      const content = fs.readFileSync(path.join(dir, file), "utf-8");
      const { slug, date, dateModified, translationKey } =
        extractFromFrontmatter(content);
      if (slug && !seen.has(slug)) {
        seen.add(slug);
        entries.push({
          slug,
          lastmod: dateModified ?? date ?? "",
          translationKey,
          hreflang: HREFLANG_BY_DIR[lang] ?? lang,
        });
      }
    }
  }
  return entries.sort((a, b) => a.slug.localeCompare(b.slug));
}

// hreflang alternates per post: all members of the post's translation
// cluster (including itself) plus x-default on the English version.
// Single-member clusters get no alternates.
function blogAlternates(
  post: BlogEntry,
  all: readonly BlogEntry[],
  siteUrl: string,
): SitemapAlternate[] | undefined {
  if (!post.translationKey) return undefined;
  const cluster = all.filter((p) => p.translationKey === post.translationKey);
  if (cluster.length < 2) return undefined;

  const alternates = cluster.map((p) => ({
    hreflang: p.hreflang,
    href: `${siteUrl}/blog/${p.slug}`,
  }));
  const english = alternates.find((a) => a.hreflang === "en");
  if (english) {
    alternates.push({ hreflang: "x-default", href: english.href });
  }
  return alternates;
}

function buildSitemapXml(entries: readonly SitemapEntry[]): string {
  const urls = entries
    .map((e) => {
      const lastmodTag = e.lastmod
        ? `\n    <lastmod>${escapeXml(e.lastmod)}</lastmod>`
        : "";
      const alternateTags = (e.alternates ?? [])
        .map(
          (a) =>
            `\n    <xhtml:link rel="alternate" hreflang="${escapeXml(a.hreflang)}" href="${escapeXml(a.href)}"/>`,
        )
        .join("");
      return `  <url>\n    <loc>${escapeXml(e.loc)}</loc>${lastmodTag}\n    <changefreq>${escapeXml(e.changefreq)}</changefreq>\n    <priority>${escapeXml(e.priority)}</priority>${alternateTags}\n  </url>`;
    })
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n${urls}\n</urlset>\n`;
}

function buildEntries(
  siteUrl: string,
  posts: readonly BlogEntry[],
  buildDate: string,
): SitemapEntry[] {
  const latestPostDate =
    posts.reduce<string>((acc, p) => (p.lastmod > acc ? p.lastmod : acc), "") ||
    buildDate;

  return [
    {
      loc: `${siteUrl}/`,
      changefreq: "weekly",
      priority: "1.0",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/features/applications`,
      changefreq: "monthly",
      priority: "0.8",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/features/resume-builder`,
      changefreq: "monthly",
      priority: "0.8",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/features/cover-letters`,
      changefreq: "monthly",
      priority: "0.8",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/blog`,
      changefreq: "weekly",
      priority: "0.8",
      lastmod: latestPostDate,
    },
    ...posts.map((p) => ({
      loc: `${siteUrl}/blog/${p.slug}`,
      changefreq: "monthly" as const,
      priority: "0.7",
      lastmod: p.lastmod || buildDate,
      alternates: blogAlternates(p, posts, siteUrl),
    })),
    {
      loc: `${siteUrl}/privacy`,
      changefreq: "yearly",
      priority: "0.3",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/terms`,
      changefreq: "yearly",
      priority: "0.3",
      lastmod: buildDate,
    },
    {
      loc: `${siteUrl}/refund`,
      changefreq: "yearly",
      priority: "0.3",
      lastmod: buildDate,
    },
  ];
}

/**
 * Writes `<outDir>/sitemap.xml` for the frontend app rooted at `root`,
 * returning whether it wrote anything.
 *
 * The blog content directory is the marker for "this root is the frontend
 * app". Without it there is nothing truthful to emit, so we skip rather than
 * drop a silently blogless sitemap into whatever directory was resolved.
 * Exported so the behaviour is testable without driving a real Vite build.
 */
export function writeSitemap(root: string, outDir: string): boolean {
  const blogDir = path.join(root, "src/content/blog");
  if (!fs.existsSync(blogDir)) {
    console.warn(
      `\x1b[33m!\x1b[0m sitemap.xml skipped: no blog content at ${blogDir}`,
    );
    return false;
  }

  const siteUrl = process.env.VITE_SITE_URL ?? "https://jobber-app.com";
  const buildDate = new Date().toISOString().slice(0, 10);
  const entries = buildEntries(siteUrl, collectBlogEntries(blogDir), buildDate);
  const xml = buildSitemapXml(entries);

  if (!fs.existsSync(outDir)) {
    fs.mkdirSync(outDir, { recursive: true });
  }

  fs.writeFileSync(path.join(outDir, "sitemap.xml"), xml, "utf-8");
  console.log(
    `\x1b[32m✓\x1b[0m sitemap.xml generated with ${entries.length} URLs`,
  );
  return true;
}

export default function sitemapPlugin(): Plugin {
  let resolvedRoot: string;
  let resolvedOutDir: string;

  return {
    name: "vite-plugin-sitemap",
    // A sitemap is a build output, and `closeBundle` also fires in serve mode
    // — including under Vitest, which points `build.outDir` at the sentinel
    // "dummy-non-existing-folder" precisely so outDir writers stay harmless.
    // Creating that directory defeated the sentinel and littered the tree on
    // every test run, so restrict the plugin to real builds.
    apply: "build",
    configResolved(config: ResolvedConfig) {
      resolvedRoot = config.root;
      resolvedOutDir = path.resolve(config.root, config.build.outDir);
    },
    closeBundle() {
      writeSitemap(resolvedRoot, resolvedOutDir);
    },
  };
}
