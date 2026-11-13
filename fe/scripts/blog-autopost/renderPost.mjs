import fs from "node:fs";
import path from "node:path";

import { BLOG_LANGS } from "./existingPosts.mjs";

/**
 * One post as the markdown file the blog loader reads, with the frontmatter
 * fields in the order the hand-written posts use.
 */
export function renderPostMarkdown({ post, lang, translationKey, date }) {
  const frontmatter = [
    `title: "${post.title}"`,
    `slug: "${post.slug}"`,
    `translationKey: "${translationKey}"`,
    `date: "${date}"`,
    `description: "${post.description}"`,
    `tags: ${JSON.stringify(post.tags)}`,
    `lang: "${lang}"`,
  ];
  return `---\n${frontmatter.join("\n")}\n---\n\n${post.content.trim()}\n`;
}

/** The file each language version of `article` is written to. */
export function articleFiles(article, { blogDir, date }) {
  return BLOG_LANGS.map((lang) => ({
    lang,
    filePath: path.join(blogDir, lang, `${article[lang].slug}.md`),
    markdown: renderPostMarkdown({
      post: article[lang],
      lang,
      translationKey: article.translationKey,
      date,
    }),
  }));
}

/**
 * Writes the files, refusing to overwrite anything: validation already rules
 * out slug collisions, so an existing file means something else is wrong.
 */
export function writeArticleFiles(files) {
  for (const { filePath, markdown } of files) {
    fs.mkdirSync(path.dirname(filePath), { recursive: true });
    fs.writeFileSync(filePath, markdown, { encoding: "utf8", flag: "wx" });
  }
}
