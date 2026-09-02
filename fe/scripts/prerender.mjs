#!/usr/bin/env node
// Static prerender for public routes.
// Spins up `vite preview` on the freshly built `dist/`, drives a headless
// Chromium through each route, and writes the post-hydration HTML back to
// `dist/<route>/index.html`. The nginx SPA fallback keeps the prerendered
// HTML for crawlers while React continues to hydrate on the client.
//
// Requires Playwright Chromium:  npx playwright install chromium

import { chromium } from "playwright";
import { preview } from "vite";
import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const DIST = path.join(ROOT, "dist");
const BLOG_ROOT = path.join(ROOT, "src/content/blog");
const BLOG_LANGS = ["en", "ru", "ua"];

const STATIC_ROUTES = [
  "/",
  "/blog",
  "/privacy",
  "/terms",
  "/refund",
  "/features/applications",
  "/features/resume-builder",
  "/features/cover-letters",
];

const VALID_SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function extractSlug(content) {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) return null;
  for (const line of match[1].split("\n")) {
    const colonIndex = line.indexOf(":");
    if (colonIndex === -1) continue;
    const key = line.slice(0, colonIndex).trim();
    if (key !== "slug") continue;
    const raw = line
      .slice(colonIndex + 1)
      .trim()
      .replace(/^"|"$/g, "");
    return VALID_SLUG.test(raw) ? raw : null;
  }
  return null;
}

async function collectBlogRoutes() {
  // Prerender every localized post. Slugs are unique per language; dedup in case
  // one is shared across languages (same /blog/<slug> URL).
  const routes = new Set();
  for (const lang of BLOG_LANGS) {
    const dir = path.join(BLOG_ROOT, lang);
    try {
      const files = await fs.readdir(dir);
      for (const file of files) {
        if (!file.endsWith(".md")) continue;
        const content = await fs.readFile(path.join(dir, file), "utf8");
        const slug = extractSlug(content);
        if (slug) routes.add(`/blog/${slug}`);
      }
    } catch {
      // language directory missing — skip
    }
  }
  return [...routes].sort();
}

function outPathFor(route) {
  if (route === "/") return path.join(DIST, "index.html");
  return path.join(DIST, route.slice(1), "index.html");
}

async function ensureDistExists() {
  try {
    await fs.access(path.join(DIST, "index.html"));
  } catch {
    console.error(
      "[prerender] dist/index.html not found — run `vite build` first.",
    );
    process.exit(1);
  }
}

async function waitForHydration(page) {
  // The home page injects a JSON-LD <script> after mount via JsonLd.tsx;
  // the same pattern is used on blog and post pages. If the script exists
  // we know React has rendered at least the head-level effects.
  await page
    .waitForFunction(
      () =>
        document.querySelectorAll('script[type="application/ld+json"]').length >
        0,
      { timeout: 8000 },
    )
    .catch(() => {});
  // Belt-and-braces: small settle so React Router transitions complete.
  await page.waitForTimeout(200);
}

async function main() {
  await ensureDistExists();

  const blogRoutes = await collectBlogRoutes();
  const routes = [...STATIC_ROUTES, ...blogRoutes];

  const server = await preview({
    root: ROOT,
    preview: { port: 4173, strictPort: true },
    logLevel: "warn",
  });
  const baseUrl = `http://localhost:${server.config.preview.port}`;

  const browser = await chromium.launch();
  const page = await browser.newPage({
    viewport: { width: 1280, height: 720 },
    userAgent:
      "Mozilla/5.0 (compatible; JobberPrerender/1.0; +https://jobber-app.com)",
  });

  // Pre-seed cookie consent so the banner is not baked into every prerendered
  // snapshot (and nothing analytics-shaped runs during builds).
  await page.addInitScript(() => {
    try {
      localStorage.setItem(
        "cookie-consent",
        JSON.stringify({ choice: "essential", at: "prerender" }),
      );
    } catch {
      // localStorage unavailable — the banner in snapshots is cosmetic only.
    }
  });

  // Render every route against the pristine SPA shell first, only then
  // write to disk. Otherwise vite preview would serve a freshly written
  // /index.html (with home-specific JSON-LD) to subsequent route renders.
  const rendered = [];
  let failed = 0;

  for (const route of routes) {
    try {
      await page.goto(`${baseUrl}${route}`, {
        waitUntil: "networkidle",
        timeout: 15000,
      });
      await waitForHydration(page);

      const html = await page.content();
      rendered.push({ route, html });
      console.log(
        `\x1b[36m·\x1b[0m rendered ${route} (${Math.round(html.length / 1024)} KB)`,
      );
    } catch (err) {
      console.error(`\x1b[31m✗\x1b[0m ${route} — ${err.message}`);
      failed++;
    }
  }

  await browser.close();
  server.httpServer.close();

  for (const { route, html } of rendered) {
    const outPath = outPathFor(route);
    await fs.mkdir(path.dirname(outPath), { recursive: true });
    await fs.writeFile(outPath, html, "utf8");
    console.log(`\x1b[32m✓\x1b[0m wrote ${path.relative(ROOT, outPath)}`);
  }

  console.log(
    `\n[prerender] ${rendered.length} routes rendered, ${failed} failed, ${routes.length} total`,
  );

  if (failed > 0) process.exit(1);
}

main().catch((err) => {
  console.error("[prerender] fatal:", err);
  process.exit(1);
});
