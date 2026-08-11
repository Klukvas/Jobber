#!/usr/bin/env node
// Renders Chrome Web Store promo assets for the browser extension with headless
// Chromium (Playwright, already a dependency here). Lives in fe/scripts so the
// bare "playwright" import resolves against fe/node_modules; outputs PNGs into
// ext/promo.
//
//   node fe/scripts/render-ext-tiles.mjs
//
// Outputs:
//   promo-tile-440x280.png     small promo tile (vertical layout)
//   promo-marquee-1400x560.png marquee promo tile (horizontal layout)

import { chromium } from "playwright";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const EXT = path.resolve(__dirname, "../../ext");
const OUT_DIR = path.join(EXT, "promo");
const ICON = fs
  .readFileSync(path.join(EXT, "icons/icon128.png"))
  .toString("base64");

const SHARED = /* css */ `
  @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@500;600;700;800&display=swap');
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family:'DM Sans',sans-serif; color:#fff; overflow:hidden; position:relative;
    background:linear-gradient(135deg,#6366f1 0%,#4f46e5 52%,#3730a3 100%); }
  .grid { position:absolute; inset:0;
    background-image:
      linear-gradient(rgba(255,255,255,.06) 1px,transparent 1px),
      linear-gradient(90deg,rgba(255,255,255,.06) 1px,transparent 1px); }
  .brand { display:flex; align-items:center; }
  .brand img { border-radius:22%; box-shadow:0 6px 18px rgba(0,0,0,.35); }
  .brand span { font-weight:800; letter-spacing:-.02em; }
  h1 { font-weight:800; letter-spacing:-.035em; line-height:1.03; }
  h1 em { font-style:normal; color:#a3e635; }
  .feat { font-weight:600; color:rgba(255,255,255,.82); }
  .pill { display:inline-flex; align-items:center; gap:7px; background:#a3e635; color:#1a1a2e;
    font-weight:800; border-radius:999px; box-shadow:0 8px 20px rgba(163,230,53,.35); }
`;

// Small promo tile — 440×280, vertical layout.
const TILE = /* html */ `<!doctype html><html><head><meta charset="utf-8"/><style>${SHARED}
  body { width:440px; height:280px; }
  .grid { background-size:34px 34px; }
  .glow { position:absolute; width:420px; height:420px; border-radius:50%; top:-190px; right:-140px;
    background:radial-gradient(circle,rgba(255,255,255,.28) 0%,transparent 62%); }
  .glow2 { position:absolute; width:300px; height:300px; border-radius:50%; bottom:-160px; left:-120px;
    background:radial-gradient(circle,rgba(163,230,53,.22) 0%,transparent 65%); }
  .wrap { position:relative; height:100%; padding:30px 32px; display:flex; flex-direction:column; }
  .brand { gap:11px; } .brand img { width:40px; height:40px; } .brand span { font-size:23px; }
  h1 { margin-top:auto; font-size:40px; }
  .feat { margin-top:13px; font-size:14.5px; }
  .pill { margin-top:16px; align-self:flex-start; font-size:13.5px; padding:8px 14px; }
  .pill b { font-size:15px; }
</style></head><body>
  <div class="grid"></div><div class="glow"></div><div class="glow2"></div>
  <div class="wrap">
    <div class="brand"><img src="data:image/png;base64,${ICON}"/><span>Jobber</span></div>
    <h1>Save any job.<br/><em>One click.</em></h1>
    <div class="feat">AI parsing · Resume match score · Autofill</div>
    <div class="pill"><b>＋</b> Save to Jobber · Free</div>
  </div>
</body></html>`;

// Marquee promo tile — 1400×560, horizontal layout with a floating "saved job" card.
const MARQUEE = /* html */ `<!doctype html><html><head><meta charset="utf-8"/><style>${SHARED}
  body { width:1400px; height:560px; }
  .grid { background-size:56px 56px; }
  .glow { position:absolute; width:900px; height:900px; border-radius:50%; top:-420px; right:-200px;
    background:radial-gradient(circle,rgba(255,255,255,.22) 0%,transparent 60%); }
  .glow2 { position:absolute; width:640px; height:640px; border-radius:50%; bottom:-360px; left:-220px;
    background:radial-gradient(circle,rgba(163,230,53,.20) 0%,transparent 65%); }
  .brand { position:absolute; top:54px; left:90px; gap:15px; z-index:2; }
  .brand img { width:56px; height:56px; } .brand span { font-size:34px; }
  .hero { position:relative; height:100%; display:flex; align-items:center; gap:70px; padding:0 90px; }
  .copy { flex:1; }
  .copy h1 { font-size:104px; }
  .copy .feat { margin-top:26px; font-size:26px; }
  .copy .pill { margin-top:34px; font-size:22px; padding:15px 28px; }
  .copy .pill b { font-size:26px; }
  .visual { width:470px; flex:none; display:flex; justify-content:center; }
  .card { width:440px; background:#fff; color:#1a1a2e; border-radius:24px; padding:30px 32px;
    box-shadow:0 50px 100px rgba(15,10,60,.45); transform:rotate(-3deg); }
  .card .src { display:inline-block; font-size:15px; font-weight:800; color:#4f46e5;
    background:#eef0ff; padding:5px 12px; border-radius:8px; }
  .card .title { font-size:30px; font-weight:800; margin-top:16px; letter-spacing:-.02em; }
  .card .meta { font-size:19px; color:#64748b; margin-top:8px; font-weight:600; }
  .card .saved { margin-top:24px; display:inline-flex; align-items:center; gap:9px;
    background:#a3e635; color:#1a1a2e; font-weight:800; font-size:20px; padding:13px 20px; border-radius:12px; }
</style></head><body>
  <div class="grid"></div><div class="glow"></div><div class="glow2"></div>
  <div class="brand"><img src="data:image/png;base64,${ICON}"/><span>Jobber</span></div>
  <div class="hero">
    <div class="copy">
      <h1>Save any job.<br/><em>One click.</em></h1>
      <div class="feat">AI parsing · Resume match score · Application autofill</div>
      <div class="pill"><b>＋</b> Save to Jobber · Free</div>
    </div>
    <div class="visual">
      <div class="card">
        <div class="src">LinkedIn</div>
        <div class="title">Senior Frontend Engineer</div>
        <div class="meta">Stripe · Remote · $180k</div>
        <div class="saved">✓ Saved to Jobber</div>
      </div>
    </div>
  </div>
</body></html>`;

const OUTPUTS = [
  { file: "promo-tile-440x280.png", html: TILE, w: 440, h: 280 },
  { file: "promo-marquee-1400x560.png", html: MARQUEE, w: 1400, h: 560 },
];

const browser = await chromium.launch();
for (const { file, html, w, h } of OUTPUTS) {
  const page = await browser.newPage({
    viewport: { width: w, height: h },
    deviceScaleFactor: 1,
  });
  await page.setContent(html, { waitUntil: "networkidle" });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(OUT_DIR, file),
    clip: { x: 0, y: 0, width: w, height: h },
  });
  await page.close();
  console.log(`  ✓ ${file} (${w}×${h})`);
}
await browser.close();
