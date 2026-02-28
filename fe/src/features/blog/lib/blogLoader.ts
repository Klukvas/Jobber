import { parseISO } from "date-fns";

export interface BlogPost {
  readonly title: string;
  readonly slug: string;
  readonly date: string;
  readonly description: string;
  readonly tags: readonly string[];
  readonly lang: string;
  readonly content: string;
}

interface FrontmatterData {
  readonly title?: string;
  readonly slug?: string;
  readonly date?: string;
  readonly description?: string;
  readonly tags?: string[];
  readonly lang?: string;
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

export function getPostBySlug(
  slug: string,
  lang: string,
): BlogPost | undefined {
  const posts = getAllPosts(lang);
  return posts.find((p) => p.slug === slug);
}
