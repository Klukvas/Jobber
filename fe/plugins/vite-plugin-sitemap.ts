import fs from "node:fs";
import path from "node:path";
import type { Plugin, ResolvedConfig } from "vite";

const VALID_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

interface SitemapEntry {
  readonly loc: string;
  readonly changefreq: string;
  readonly priority: string;
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
function extractSlugFromFrontmatter(content: string): string | null {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) return null;

  for (const line of match[1].split("\n")) {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) continue;
    const key = line.slice(0, colonIndex).trim();
    if (key === "slug") {
      const raw = line
        .slice(colonIndex + 1)
        .trim()
        .replace(/^"|"$/g, "");
      return VALID_SLUG.test(raw) ? raw : null;
    }
  }
  return null;
}

function collectBlogSlugs(blogDir: string): string[] {
  const enDir = path.join(blogDir, "en");
  if (!fs.existsSync(enDir)) return [];

  const slugs: string[] = [];
  for (const file of fs.readdirSync(enDir)) {
    if (!file.endsWith(".md")) continue;
    const content = fs.readFileSync(path.join(enDir, file), "utf-8");
    const slug = extractSlugFromFrontmatter(content);
    if (slug) {
      slugs.push(slug);
    }
  }
  return slugs.sort();
}

function buildSitemapXml(entries: readonly SitemapEntry[]): string {
  const urls = entries
    .map(
      (e) =>
        `  <url>\n    <loc>${escapeXml(e.loc)}</loc>\n    <changefreq>${escapeXml(e.changefreq)}</changefreq>\n    <priority>${escapeXml(e.priority)}</priority>\n  </url>`,
    )
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
      const slugs = collectBlogSlugs(blogDir);

      const entries: SitemapEntry[] = [
        { loc: `${siteUrl}/`, changefreq: "weekly", priority: "1.0" },
        {
          loc: `${siteUrl}/features/applications`,
          changefreq: "monthly",
          priority: "0.8",
        },
        {
          loc: `${siteUrl}/features/resume-builder`,
          changefreq: "monthly",
          priority: "0.8",
        },
        {
          loc: `${siteUrl}/features/cover-letters`,
          changefreq: "monthly",
          priority: "0.8",
        },
        { loc: `${siteUrl}/blog`, changefreq: "weekly", priority: "0.8" },
        ...slugs.map((slug) => ({
          loc: `${siteUrl}/blog/${slug}`,
          changefreq: "monthly" as const,
          priority: "0.7",
        })),
        { loc: `${siteUrl}/privacy`, changefreq: "yearly", priority: "0.3" },
        { loc: `${siteUrl}/terms`, changefreq: "yearly", priority: "0.3" },
        { loc: `${siteUrl}/refund`, changefreq: "yearly", priority: "0.3" },
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
