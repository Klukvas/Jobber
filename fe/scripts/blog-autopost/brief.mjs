#!/usr/bin/env node
/**
 * Prints everything the scheduled Claude routine needs before writing a post:
 * the editorial brief, the articles already published and the file format
 * publish.mjs accepts.
 *
 *   node scripts/blog-autopost/brief.mjs
 */
import fs from "node:fs";

import { readExistingPosts } from "./existingPosts.mjs";
import { BLOG_DIR, PRODUCT_FACTS_PATH, publicationDateToday } from "./paths.mjs";
import {
  buildEditorialBrief,
  buildTopicRequest,
  describeArticleFile,
} from "./prompt.mjs";

const existingPosts = readExistingPosts(BLOG_DIR);

console.log(
  [
    buildEditorialBrief(fs.readFileSync(PRODUCT_FACTS_PATH, "utf8")),
    buildTopicRequest({ existingPosts, date: publicationDateToday() }),
    describeArticleFile(),
  ].join("\n\n"),
);
