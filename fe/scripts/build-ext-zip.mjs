#!/usr/bin/env node
// Builds a clean Chrome Web Store zip from ext/. The source tree keeps dev-only
// localhost origins for local testing; this strips them from a throwaway copy so
// the published package only talks to https://jobber-app.com. Excludes promo/
// assets and dev tooling. Version + description come from ext/manifest.json.
//
//   node fe/scripts/build-ext-zip.mjs

import fs from "node:fs";
import path from "node:path";
import { execSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO = path.resolve(__dirname, "../..");
const EXT = path.join(REPO, "ext");
const BUILD = path.join("/tmp", "jobber-ext-build");

// Only the actual extension files — NOT promo/ (dashboard assets) or dotfiles.
const INCLUDE = [
  "manifest.json",
  "background.js",
  "content",
  "popup",
  "sidepanel",
  "shared",
  "icons",
];

fs.rmSync(BUILD, { recursive: true, force: true });
fs.mkdirSync(BUILD, { recursive: true });
for (const item of INCLUDE) {
  fs.cpSync(path.join(EXT, item), path.join(BUILD, item), { recursive: true });
}

// ── Strip localhost ──────────────────────────────────────
const ORIGINS_DEV = `["https://jobber-app.com", "http://localhost:8080"]`;
const ORIGINS_PROD = `["https://jobber-app.com"]`;

for (const rel of ["shared/api.js", "background.js"]) {
  const p = path.join(BUILD, rel);
  fs.writeFileSync(p, fs.readFileSync(p, "utf8").split(ORIGINS_DEV).join(ORIGINS_PROD));
}

// Collapse the dev-only getWebAppBase branch to the prod one.
const apiPath = path.join(BUILD, "shared/api.js");
fs.writeFileSync(
  apiPath,
  fs.readFileSync(apiPath, "utf8").replace(
    /    \/\/ In production[\s\S]*?    return _apiBase \|\| API_BASE;/,
    "    return _apiBase || API_BASE;",
  ),
);

// Manifest: drop localhost host permissions (re-serialize for a clean file).
const manifestPath = path.join(BUILD, "manifest.json");
const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
manifest.host_permissions = manifest.host_permissions.filter(
  (h) => !h.includes("localhost"),
);
fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + "\n");

// ── Guard: nothing dev-only leaked ───────────────────────
const leaked = execSync(`grep -rn localhost ${BUILD} || true`).toString().trim();
if (leaked) {
  console.error("✗ localhost still present in build:\n" + leaked);
  process.exit(1);
}

// ── Zip ──────────────────────────────────────────────────
const version = manifest.version;
const zipName = `jobber-extension-v${version}.zip`;
const zipPath = path.join(REPO, zipName);
fs.rmSync(zipPath, { force: true });
execSync(`cd ${BUILD} && zip -rq ${zipPath} . -x '*.DS_Store'`);

console.log(`✓ built ${zipName} (v${version})`);
console.log(`  files: ${INCLUDE.join(", ")}`);
console.log(`  localhost stripped, promo/ excluded`);
