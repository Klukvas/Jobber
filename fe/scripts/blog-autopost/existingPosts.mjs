import fs from "node:fs";
import path from "node:path";

/** Content directories under `src/content/blog`, one per language. */
export const BLOG_LANGS = /** @type {const} */ (["en", "ua", "ru"]);

/**
 * Reads the single-line frontmatter the blog uses.
 *
 * Mirrors the parser in `src/features/blog/lib/blogLoader.ts` (and the sitemap
 * plugin's copy): `key: value` per line, surrounding double quotes stripped
 * without unescaping, JSON arrays for lists. Keep the three in sync.
 */
export function parseFrontmatter(raw) {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/);
  if (!match) return {};

  const entries = match[1].split("\n").flatMap((line) => {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) return [];
    const key = line.slice(0, colonIndex).trim();
    return [[key, parseValue(line.slice(colonIndex + 1).trim())]];
  });
  return Object.fromEntries(entries);
}

function parseValue(rawValue) {
  const unquoted =
    rawValue.startsWith('"') && rawValue.endsWith('"')
      ? rawValue.slice(1, -1)
      : rawValue;
  if (!unquoted.startsWith("[")) return unquoted;
  try {
    return JSON.parse(unquoted);
  } catch {
    // The app keeps a malformed array as its raw string; so do we.
    return unquoted;
  }
}

/**
 * Every published post's metadata, across all language directories.
 * A missing language directory is treated as empty rather than an error.
 */
export function readExistingPosts(blogDir) {
  return BLOG_LANGS.flatMap((lang) => {
    const dir = path.join(blogDir, lang);
    if (!fs.existsSync(dir)) return [];
    return fs
      .readdirSync(dir)
      .filter((file) => file.endsWith(".md"))
      .sort()
      .map((file) => {
        const data = parseFrontmatter(
          fs.readFileSync(path.join(dir, file), "utf8"),
        );
        return {
          lang,
          file,
          title: String(data.title ?? ""),
          slug: String(data.slug ?? path.basename(file, ".md")),
          translationKey: data.translationKey
            ? String(data.translationKey)
            : undefined,
          date: String(data.date ?? ""),
          tags: Array.isArray(data.tags) ? data.tags.map(String) : [],
        };
      });
  });
}
