import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { loadPosts } from "../../src/features/blog/lib/blogLoader";
import { validateArticle } from "../blog-autopost/articleSchema.mjs";
import {
  parseFrontmatter,
  readExistingPosts,
} from "../blog-autopost/existingPosts.mjs";
import { buildTopicRequest } from "../blog-autopost/prompt.mjs";
import {
  articleFiles,
  renderPostMarkdown,
  writeArticleFiles,
} from "../blog-autopost/renderPost.mjs";

const realBlogDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../src/content/blog",
);

function body(title, sentence, repeat = 50) {
  return `# ${title}\n\n## Section\n\n${Array(repeat).fill(sentence).join(" ")}`;
}

function validArticle() {
  return {
    translationKey: "follow-up-after-interview",
    en: {
      title: "How to Follow Up After an Interview Without Being Pushy",
      slug: "follow-up-after-interview",
      description:
        "When to send a follow-up after an interview, what to write, and how to keep track of every thread so nothing slips.",
      tags: ["interview", "follow-up", "job search"],
      content: body(
        "How to Follow Up After an Interview Without Being Pushy",
        "Track every application in one place so you can follow up on time. See https://jobber-app.com/features/applications for a board.",
      ),
    },
    ua: {
      title: "Як нагадати про себе після співбесіди",
      slug: "yak-nahadaty-pro-sebe-pislya-spivbesidy",
      description:
        "Коли надсилати лист після співбесіди, що в ньому писати і як не загубити жодної розмови з рекрутером.",
      tags: ["співбесіда", "пошук роботи"],
      content: body(
        "Як нагадати про себе після співбесіди",
        "Відстежуйте кожен відгук в одному місці, щоб вчасно нагадати про себе.",
      ),
    },
    ru: {
      title: "Как напомнить о себе после собеседования",
      slug: "kak-napomnit-o-sebe-posle-sobesedovaniya",
      description:
        "Когда отправлять письмо после собеседования, что в нём писать и как не потерять ни одной переписки с рекрутером.",
      tags: ["собеседование", "поиск работы"],
      content: body(
        "Как напомнить о себе после собеседования",
        "Отслеживайте каждый отклик в одном месте, чтобы вовремя напомнить о себе.",
      ),
    },
  };
}

function withPost(article, lang, changes) {
  return { ...article, [lang]: { ...article[lang], ...changes } };
}

describe("validateArticle", () => {
  const existing = readExistingPosts(realBlogDir);

  it("accepts a complete article that collides with nothing", () => {
    expect(validateArticle(validArticle(), existing)).toEqual([]);
  });

  it("rejects a slug or translationKey that is already published", () => {
    const taken = existing.find((post) => post.translationKey);
    const article = {
      ...withPost(validArticle(), "en", { slug: taken.slug }),
      translationKey: taken.translationKey,
    };

    const errors = validateArticle(article, existing);

    expect(errors).toContain(`slug "${taken.slug}" is already used by a published post`);
    expect(errors.some((e) => e.startsWith(`translationKey "${taken.translationKey}"`))).toBe(true);
  });

  it("rejects values the frontmatter parser cannot hold", () => {
    const article = withPost(validArticle(), "en", {
      title: 'The "Perfect" Resume',
      description: "line one\nline two that is long enough to pass the length check on its own merits.",
    });

    const errors = validateArticle(article, existing);

    expect(errors.some((e) => e.startsWith("en.title must not contain straight double quotes"))).toBe(true);
    expect(errors).toContain("en.description must be a single line");
  });

  it("rejects links to any host other than the site", () => {
    const article = withPost(validArticle(), "en", {
      content: `${validArticle().en.content} [study](https://example.org/fake-study)`,
    });

    expect(validateArticle(article, existing)).toContain(
      "en.content links to example.org; only jobber-app.com links are allowed",
    );
  });

  it("rejects a Russian body filed as Ukrainian", () => {
    const article = withPost(validArticle(), "ua", { content: validArticle().ru.content });

    expect(validateArticle(article, existing)).toContain(
      "ua.content does not read as Ukrainian (no і/ї/є/ґ)",
    );
  });

  it("rejects a body that is too short or lacks its heading", () => {
    const article = withPost(validArticle(), "en", { content: "Just a few words." });

    const errors = validateArticle(article, existing);

    expect(errors).toContain('en.content must start with a "# " heading');
    expect(errors.some((e) => e.startsWith("en.content has 4 words"))).toBe(true);
  });
});

describe("renderPostMarkdown", () => {
  it("writes frontmatter the blog loader reads back unchanged", () => {
    const article = validArticle();
    const markdown = renderPostMarkdown({
      post: article.ua,
      lang: "ua",
      translationKey: article.translationKey,
      date: "2026-11-17",
    });

    const [loaded] = loadPosts({ "/post.md": markdown });

    expect(loaded).toMatchObject({
      title: article.ua.title,
      slug: article.ua.slug,
      translationKey: article.translationKey,
      date: "2026-11-17",
      description: article.ua.description,
      tags: article.ua.tags,
      lang: "ua",
    });
    expect(loaded.content).toBe(article.ua.content);
  });
});

describe("writing and re-reading posts", () => {
  let blogDir;

  beforeEach(() => {
    blogDir = fs.mkdtempSync(path.join(os.tmpdir(), "blog-autopost-"));
  });

  afterEach(() => {
    fs.rmSync(blogDir, { recursive: true, force: true });
  });

  it("writes one file per language that readExistingPosts then sees", () => {
    const files = articleFiles(validArticle(), { blogDir, date: "2026-11-17" });
    writeArticleFiles(files);

    const posts = readExistingPosts(blogDir);

    expect(posts.map((post) => [post.lang, post.slug, post.translationKey])).toEqual([
      ["en", "follow-up-after-interview", "follow-up-after-interview"],
      ["ua", "yak-nahadaty-pro-sebe-pislya-spivbesidy", "follow-up-after-interview"],
      ["ru", "kak-napomnit-o-sebe-posle-sobesedovaniya", "follow-up-after-interview"],
    ]);
  });

  it("refuses to overwrite an existing post", () => {
    const files = articleFiles(validArticle(), { blogDir, date: "2026-11-17" });
    writeArticleFiles(files);

    expect(() => writeArticleFiles(files)).toThrow(/EEXIST/);
  });
});

describe("existing content", () => {
  it("parses every published post with a slug and a date", () => {
    const posts = readExistingPosts(realBlogDir);

    expect(posts.length).toBeGreaterThan(0);
    for (const post of posts) {
      expect(post.slug, post.file).not.toBe("");
      expect(post.date, post.file).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    }
  });

  it("lists every published article in the prompt so topics are not repeated", () => {
    const posts = readExistingPosts(realBlogDir);
    const prompt = buildTopicRequest({ existingPosts: posts, date: "2026-11-17" });

    for (const post of posts) {
      expect(prompt).toContain(post.title);
    }
  });

  it("strips quotes and parses JSON arrays like the app does", () => {
    expect(
      parseFrontmatter('---\ntitle: "A: B"\ntags: ["x", "y"]\n---\nbody'),
    ).toEqual({ title: "A: B", tags: ["x", "y"] });
  });
});

describe("publish.mjs", () => {
  const script = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../blog-autopost/publish.mjs");
  let workDir;

  beforeEach(() => {
    workDir = fs.mkdtempSync(path.join(os.tmpdir(), "blog-publish-"));
  });

  afterEach(() => {
    fs.rmSync(workDir, { recursive: true, force: true });
  });

  function runCheck(article) {
    const file = path.join(workDir, "article.json");
    fs.writeFileSync(file, JSON.stringify(article));
    return spawnSync(process.execPath, [script, file, "--check", "--date", "2099-01-01"], {
      encoding: "utf8",
    });
  }

  it("accepts a valid article without writing anything", () => {
    const result = runCheck(validArticle());

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("en/follow-up-after-interview.md");
    expect(fs.existsSync(path.join(realBlogDir, "en/follow-up-after-interview.md"))).toBe(false);
  });

  it("exits 1 and lists the problems for an article that needs fixing", () => {
    const result = runCheck(withPost(validArticle(), "en", { slug: "Not A Slug" }));

    expect(result.status).toBe(1);
    expect(result.stderr).toContain("en.slug");
  });
});
