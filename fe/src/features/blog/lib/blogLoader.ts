import { parseISO } from "date-fns";

export interface BlogPost {
  readonly title: string;
  readonly slug: string;
  readonly date: string;
  readonly dateModified?: string;
  readonly description: string;
  readonly tags: readonly string[];
  readonly lang: string;
  readonly image?: string;
  readonly content: string;
  // Shared id linking translations of the same article across languages.
  // Posts with the same translationKey form one hreflang cluster.
  readonly translationKey?: string;
}

interface FrontmatterData {
  readonly title?: string;
  readonly slug?: string;
  readonly date?: string;
  readonly dateModified?: string;
  readonly description?: string;
  readonly tags?: string[];
  readonly lang?: string;
  readonly image?: string;
  readonly translationKey?: string;
}

// Parses single-line YAML frontmatter only.
// Arrays must use JSON syntax: tags: ["a", "b"] — YAML block sequences are not supported.
// Dates must be ISO 8601 format: YYYY-MM-DD.
function parseFrontmatter(raw: string): {
  data: FrontmatterData;
  content: string;
} {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/);
  if (!match) {
    return { data: {}, content: raw };
  }

  const [, frontmatterBlock, content] = match;
  const data: Record<string, unknown> = {};

  for (const line of frontmatterBlock.split("\n")) {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) continue;

    const key = line.slice(0, colonIndex).trim();
    let value: unknown = line.slice(colonIndex + 1).trim();

    // Remove surrounding quotes
    if (
      typeof value === "string" &&
      value.startsWith('"') &&
      value.endsWith('"')
    ) {
      value = value.slice(1, -1);
    }

    // Parse arrays like ["a", "b"]
    if (typeof value === "string" && value.startsWith("[")) {
      try {
        value = JSON.parse(value);
      } catch {
        // keep as string
      }
    }

    data[key] = value;
  }

  return { data: data as FrontmatterData, content: content.trim() };
}

const enModules = import.meta.glob("/src/content/blog/en/*.md", {
  query: "?raw",
  eager: true,
  import: "default",
}) as Record<string, string>;

const uaModules = import.meta.glob("/src/content/blog/ua/*.md", {
  query: "?raw",
  eager: true,
  import: "default",
}) as Record<string, string>;

const ruModules = import.meta.glob("/src/content/blog/ru/*.md", {
  query: "?raw",
  eager: true,
  import: "default",
}) as Record<string, string>;

function loadPosts(modules: Record<string, string>): readonly BlogPost[] {
  return Object.values(modules)
    .map((raw) => {
      const { data, content } = parseFrontmatter(raw);
      return {
        title: data.title ?? "",
        slug: data.slug ?? "",
        date: data.date ?? "",
        description: data.description ?? "",
        tags: data.tags ?? [],
        lang: data.lang ?? "en",
        content,
        translationKey: data.translationKey,
      } satisfies BlogPost;
    })
    .sort((a, b) => {
      const bTime = b.date ? parseISO(b.date).getTime() : 0;
      const aTime = a.date ? parseISO(a.date).getTime() : 0;
      return (
        (Number.isNaN(bTime) ? 0 : bTime) - (Number.isNaN(aTime) ? 0 : aTime)
      );
    });
}

const enPosts = loadPosts(enModules);
const uaPosts = loadPosts(uaModules);
const ruPosts = loadPosts(ruModules);

export function getAllPosts(lang: string): readonly BlogPost[] {
  if (lang === "ua") return uaPosts;
  if (lang === "ru") return ruPosts;
  return enPosts;
}

const ALL_LANGS = ["en", "ua", "ru"] as const;

export interface HreflangAlternate {
  // BCP 47 code for the hreflang attribute. The "ua" content directory holds
  // Ukrainian, whose language code is "uk" — never emit "ua".
  readonly hreflang: string;
  readonly slug: string;
}

const HREFLANG_BY_DIR = { en: "en", ua: "uk", ru: "ru" } as const;

// Translations of `post` across languages (including the post itself),
// ordered en → uk → ru. Returns [] when the post has no translationKey or
// no counterpart in another language — a single-entry cluster is meaningless.
export function getHreflangAlternates(
  post: BlogPost,
): readonly HreflangAlternate[] {
  if (!post.translationKey) return [];
  const alternates: HreflangAlternate[] = [];
  for (const dir of ALL_LANGS) {
    const match = getAllPosts(dir).find(
      (p) => p.translationKey === post.translationKey,
    );
    if (match) {
      alternates.push({ hreflang: HREFLANG_BY_DIR[dir], slug: match.slug });
    }
  }
  return alternates.length > 1 ? alternates : [];
}

export function getPostBySlug(
  slug: string,
  lang: string,
): BlogPost | undefined {
  // Prefer the requested language, then fall back to any language. Localized
  // posts have unique slugs, so a RU/UA URL still resolves when the UI language
  // is English — e.g. a crawler (or shared link) hitting the localized URL.
  const inLang = getAllPosts(lang).find((p) => p.slug === slug);
  if (inLang) return inLang;
  for (const l of ALL_LANGS) {
    if (l === lang) continue;
    const found = getAllPosts(l).find((p) => p.slug === slug);
    if (found) return found;
  }
  return undefined;
}
