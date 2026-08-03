import fs from "node:fs";
import path from "node:path";
import type { Plugin, ResolvedConfig } from "vite";

const VALID_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const VALID_DATE = /^\d{4}-\d{2}-\d{2}$/;

interface SitemapEntry {
  readonly loc: string;
  readonly changefreq: string;
  readonly priority: string;
  readonly lastmod?: string;
}

interface BlogEntry {
  readonly slug: string;
  readonly lastmod: string;
}

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
} {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) return { slug: null, date: null, dateModified: null };

  let slug: string | null = null;
  let date: string | null = null;
  let dateModified: string | null = null;

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
  }

  return { slug, date, dateModified };
}

function collectBlogEntries(blogDir: string): BlogEntry[] {
  const enDir = path.join(blogDir, "en");
  if (!fs.existsSync(enDir)) return [];

  const entries: BlogEntry[] = [];
  for (const file of fs.readdirSync(enDir)) {
    if (!file.endsWith(".md")) continue;
    const content = fs.readFileSync(path.join(enDir, file), "utf-8");
    const { slug, date, dateModified } = extractFromFrontmatter(content);
    if (slug) {
      entries.push({ slug, lastmod: dateModified ?? date ?? "" });
    }
  }
  return entries.sort((a, b) => a.slug.localeCompare(b.slug));
}

function buildSitemapXml(entries: readonly SitemapEntry[]): string {
  const urls = entries
    .map((e) => {
      const lastmodTag = e.lastmod
        ? `\n    <lastmod>${escapeXml(e.lastmod)}</lastmod>`
        : "";
      return `  <url>\n    <loc>${escapeXml(e.loc)}</loc>${lastmodTag}\n    <changefreq>${escapeXml(e.changefreq)}</changefreq>\n    <priority>${escapeXml(e.priority)}</priority>\n  </url>`;
    })
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`;
}

export default function sitemapPlugin(): Plugin {
  let resolvedRoot: string;
  let resolvedOutDir: string;

  return {
    name: "vite-plugin-sitemap",
    configResolved(config: ResolvedConfig) {
      resolvedRoot = config.root;
      resolvedOutDir = path.resolve(config.root, config.build.outDir);
    },
    closeBundle() {
      const siteUrl = process.env.VITE_SITE_URL ?? "https://jobber-app.com";
      const blogDir = path.join(resolvedRoot, "src/content/blog");
      const posts = collectBlogEntries(blogDir);
      const buildDate = new Date().toISOString().slice(0, 10);
      const latestPostDate =
        posts.reduce<string>(
          (acc, p) => (p.lastmod > acc ? p.lastmod : acc),
          "",
        ) || buildDate;

      const entries: SitemapEntry[] = [
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

      const xml = buildSitemapXml(entries);

      if (!fs.existsSync(resolvedOutDir)) {
        fs.mkdirSync(resolvedOutDir, { recursive: true });
      }

      const outPath = path.join(resolvedOutDir, "sitemap.xml");
      fs.writeFileSync(outPath, xml, "utf-8");
      console.log(
        `\x1b[32m✓\x1b[0m sitemap.xml generated with ${entries.length} URLs`,
      );
    },
  };
}
