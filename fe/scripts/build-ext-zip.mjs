#!/usr/bin/env node
// Builds a clean Chrome Web Store zip from ext/. Only the reviewed runtime files
// are packaged (no tests, fixtures or promo assets). The source tree keeps
// dev-only localhost origins for local testing; they are stripped from a
// throwaway staging copy so the published package only talks to
// https://jobber-app.com. Version comes from ext/manifest.json.
//
//   node fe/scripts/build-ext-zip.mjs
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const REPO = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
export const RUNTIME_FILES = [
  "manifest.json", "background.js", "shared/api.js", "shared/auth-worker.js",
  "content/autofill.js", "content/autofill.css", "content/jobboards.js", "content/jobboards.css",
  "sidepanel/sidepanel.html", "sidepanel/sidepanel.js", "sidepanel/autofill.js", "sidepanel/sidepanel.css",
  "icons/icon16.png", "icons/icon48.png", "icons/icon128.png",
];
export function productionSource(relative, buffer) {
  if (relative === "manifest.json") {
    const manifest = JSON.parse(buffer.toString());
    manifest.host_permissions = manifest.host_permissions.filter(origin => !origin.includes("localhost"));
    return Buffer.from(JSON.stringify(manifest, null, 2) + "\n");
  }
  if (!relative.endsWith(".js")) return buffer;
  let source = buffer.toString().replace('["https://jobber-app.com", "http://localhost:8080"]', '["https://jobber-app.com"]');
  if (relative === "shared/api.js") source = source.replace(
    /    \/\/ In production[\s\S]*?    return _apiBase \|\| API_BASE;/,
    "    return _apiBase || API_BASE;",
  );
  if (/localhost/.test(source)) throw new Error(`Development origin remains in ${relative}`);
  return Buffer.from(source);
}
export function buildExtension({ outputDir = REPO } = {}) {
  const manifest = JSON.parse(fs.readFileSync(path.join(REPO, "ext/manifest.json")));
  const output = path.join(outputDir, `cursopus-extension-v${manifest.version}.zip`);
  const staging = fs.mkdtempSync(path.join(os.tmpdir(), "cursopus-ext-build-"));
  const temporaryZip = path.join(staging, "package.zip");
  const filesRoot = path.join(staging, "files");
  try {
    for (const relative of RUNTIME_FILES) {
      const destination = path.join(filesRoot, relative);
      fs.mkdirSync(path.dirname(destination), { recursive: true });
      fs.writeFileSync(destination, productionSource(relative, fs.readFileSync(path.join(REPO, "ext", relative))));
    }
    execFileSync("zip", ["-q", temporaryZip, ...RUNTIME_FILES], { cwd: filesRoot });
    // Replace the previous zip only after staging and zipping succeeded.
    fs.copyFileSync(temporaryZip, output);
    return output;
  } finally {
    // This unique directory was created above and contains only this run's files.
    fs.rmSync(staging, { recursive: true });
  }
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.stdout.write(`Built ${buildExtension()}\n`);
}
