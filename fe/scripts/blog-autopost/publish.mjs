#!/usr/bin/env node
/**
 * Validates an article JSON file and writes it as three blog posts (en, ua,
 * ru) under src/content/blog. Used by the scheduled Claude routine.
 *
 *   node scripts/blog-autopost/publish.mjs <article.json> [--check] [--force] [--date YYYY-MM-DD]
 *
 *   --check  only validate; print problems and write nothing
 *   --force  publish even if a post is already dated today
 *   --date   publication date (default: today in Kyiv)
 *
 * Exit codes: 0 written (or valid with --check), 1 the article needs fixing,
 * 2 nothing to do because a post is already dated today.
 */
import fs from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import { GeneratedArticleSchema, validateArticle } from "./articleSchema.mjs";
import { readExistingPosts } from "./existingPosts.mjs";
import { BLOG_DIR, FE_ROOT, publicationDateToday } from "./paths.mjs";
import { articleFiles, writeArticleFiles } from "./renderPost.mjs";

const EXIT_NEEDS_FIXING = 1;
const EXIT_ALREADY_PUBLISHED = 2;

function readArticle(filePath) {
  let json;
  try {
    json = JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    return { errors: [`cannot read ${filePath} as JSON: ${error.message}`] };
  }
  const parsed = GeneratedArticleSchema.safeParse(json);
  if (!parsed.success) {
    return {
      errors: parsed.error.issues.map(
        (issue) => `${issue.path.join(".") || "(root)"}: ${issue.message}`,
      ),
    };
  }
  return { article: parsed.data, errors: [] };
}

function main() {
  const { values: options, positionals } = parseArgs({
    allowPositionals: true,
    options: {
      check: { type: "boolean", default: false },
      force: { type: "boolean", default: false },
      date: { type: "string" },
    },
  });
  if (positionals.length !== 1) {
    throw new Error("usage: publish.mjs <article.json> [--check] [--force] [--date YYYY-MM-DD]");
  }
  const date = options.date ?? publicationDateToday();
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) {
    throw new Error(`--date must be YYYY-MM-DD, got "${date}"`);
  }

  const existingPosts = readExistingPosts(BLOG_DIR);
  if (!options.force && existingPosts.some((post) => post.date === date)) {
    console.log(`A post dated ${date} already exists; nothing to publish (use --force to add another).`);
    return EXIT_ALREADY_PUBLISHED;
  }

  const { article, errors: readErrors } = readArticle(positionals[0]);
  const errors = article ? validateArticle(article, existingPosts) : readErrors;
  if (errors.length > 0) {
    console.error(`The article cannot be published yet. Fix these problems:\n- ${errors.join("\n- ")}`);
    return EXIT_NEEDS_FIXING;
  }

  const files = articleFiles(article, { blogDir: BLOG_DIR, date });
  if (options.check) {
    console.log(`The article is valid. It would be written to:\n${files.map(({ filePath }) => `- ${path.relative(FE_ROOT, filePath)}`).join("\n")}`);
    return 0;
  }
  writeArticleFiles(files);
  console.log(`Published "${article.en.title}":\n${files.map(({ filePath }) => `- ${path.relative(FE_ROOT, filePath)}`).join("\n")}`);
  return 0;
}

try {
  process.exitCode = main();
} catch (error) {
  console.error(`Blog publish failed: ${error.message}`);
  process.exitCode = EXIT_NEEDS_FIXING;
}
