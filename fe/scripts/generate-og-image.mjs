#!/usr/bin/env node
// Generates public/og-image.png (1200×630) — the social share card
// referenced by og:image / twitter:image in index.html and by blog
// JSON-LD as the fallback article image.
//
// Renders an inline HTML template with headless Chromium (Playwright is
// already a dependency for scripts/prerender.mjs) and screenshots it.
//
// Usage: node scripts/generate-og-image.mjs

import { chromium } from "playwright";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const OUT_PATH = path.resolve(__dirname, "../public/og-image.png");

const WIDTH = 1200;
const HEIGHT = 630;

const TEMPLATE = /* html */ `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<style>
  @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&family=Syne:wght@800&display=swap');

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    width: ${WIDTH}px;
    height: ${HEIGHT}px;
    background: #0b0f14;
    font-family: 'DM Sans', sans-serif;
    color: #f1f5f9;
    overflow: hidden;
    position: relative;
  }

  .grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(148, 163, 184, 0.07) 1px, transparent 1px),
      linear-gradient(90deg, rgba(148, 163, 184, 0.07) 1px, transparent 1px);
    background-size: 48px 48px;
  }

  .glow {
    position: absolute;
    width: 720px;
    height: 720px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(163, 230, 53, 0.18) 0%, transparent 65%);
    top: -260px;
    right: -160px;
  }

  .content {
    position: relative;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: 0 88px;
  }

  .wordmark {
    font-family: 'Syne', sans-serif;
    font-weight: 800;
    font-size: 44px;
    letter-spacing: -0.02em;
    color: #a3e635;
    margin-bottom: 40px;
  }

  h1 {
    font-family: 'Syne', sans-serif;
    font-weight: 800;
    font-size: 64px;
    line-height: 1.08;
    letter-spacing: -0.03em;
    max-width: 980px;
    margin-bottom: 28px;
  }

  h1 .accent { color: #a3e635; }

  .subtitle {
    font-size: 28px;
    font-weight: 500;
    color: #94a3b8;
    max-width: 640px;
    line-height: 1.4;
  }

  .board {
    position: absolute;
    right: 72px;
    bottom: 64px;
    display: flex;
    gap: 14px;
  }

  .col {
    width: 120px;
    border-radius: 12px;
    background: rgba(30, 41, 59, 0.85);
    border: 1px solid rgba(148, 163, 184, 0.18);
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .card {
    height: 26px;
    border-radius: 6px;
    background: rgba(148, 163, 184, 0.22);
  }

  .card.lime { background: rgba(163, 230, 53, 0.75); }

  .col-label {
    height: 8px;
    width: 60%;
    border-radius: 4px;
    background: rgba(148, 163, 184, 0.35);
    margin-bottom: 2px;
  }
</style>
</head>
<body>
  <div class="grid"></div>
  <div class="glow"></div>
  <div class="content">
    <div class="wordmark">Jobber</div>
    <h1>Your job search,<br /><span class="accent">finally organized</span></h1>
    <div class="subtitle">AI-powered job application tracker with a Kanban board, resume builder and cover letter generator</div>
  </div>
  <div class="board">
    <div class="col">
      <div class="col-label"></div>
      <div class="card"></div>
      <div class="card"></div>
      <div class="card"></div>
    </div>
    <div class="col">
      <div class="col-label"></div>
      <div class="card lime"></div>
      <div class="card"></div>
    </div>
    <div class="col">
      <div class="col-label"></div>
      <div class="card lime"></div>
    </div>
  </div>
</body>
</html>`;

async function main() {
  const browser = await chromium.launch();
  try {
    const page = await browser.newPage({
      viewport: { width: WIDTH, height: HEIGHT },
      deviceScaleFactor: 1,
    });
    await page.setContent(TEMPLATE, { waitUntil: "networkidle" });
    // Let web fonts finish rasterizing before the screenshot.
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: OUT_PATH, type: "png" });
    console.log(`✓ wrote ${OUT_PATH}`);
  } finally {
    await browser.close();
  }
}

main().catch((err) => {
  console.error("[generate-og-image] fatal:", err);
  process.exit(1);
});
