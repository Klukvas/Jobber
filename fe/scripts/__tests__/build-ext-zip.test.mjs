// @vitest-environment node
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

import {
  buildExtension,
  productionSource,
  RUNTIME_FILES,
} from "../build-ext-zip.mjs";

const EXT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../ext");

/**
 * The store package is built from an explicit allowlist because ext/ now also
 * holds node tests and fixtures next to the runtime files — a directory copy
 * would ship them. This builds a fresh zip and checks it holds exactly the
 * transformed runtime sources, no dev origin, and every file the manifest and
 * side panel reference.
 */
describe("the Chrome Web Store zip", () => {
  let outputDir;
  let archive;
  let files;
  const unzip = (...args) => execFileSync("unzip", [...args]);

  beforeAll(() => {
    outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "cursopus-ext-zip-test-"));
    archive = buildExtension({ outputDir });
    files = unzip("-Z1", archive).toString().trim().split("\n");
  });
  afterAll(() => fs.rmSync(outputDir, { recursive: true, force: true }));

  it("contains exactly the transformed runtime files", () => {
    expect([...files].sort()).toEqual([...RUNTIME_FILES].sort());
    for (const file of RUNTIME_FILES) {
      const archived = unzip("-p", archive, file);
      const expected = productionSource(file, fs.readFileSync(path.join(EXT, file)));
      expect(archived.equals(expected), file).toBe(true);
      if (!file.endsWith(".png")) {
        expect(archived.toString()).not.toMatch(/localhost|\.test\.|promo\//);
      }
    }
  });

  it("ships a manifest that only talks to the production API", () => {
    const packaged = JSON.parse(unzip("-p", archive, "manifest.json"));
    expect(packaged.minimum_chrome_version).toBe("116");
    expect(packaged.host_permissions).toEqual(["https://cursopus.com/*"]);
  });

  it("includes every file the manifest and side panel reference", () => {
    const packaged = JSON.parse(unzip("-p", archive, "manifest.json"));
    const referenced = [
      packaged.background.service_worker,
      packaged.side_panel.default_path,
      ...Object.values(packaged.icons),
      ...packaged.content_scripts.flatMap((entry) => [...entry.js, ...entry.css]),
    ];
    const html = unzip("-p", archive, "sidepanel/sidepanel.html").toString();
    for (const match of html.matchAll(/(?:src|href)="([^"#]+\.(?:js|css))"/g)) {
      referenced.push(path.posix.normalize(`sidepanel/${match[1]}`));
    }
    for (const file of referenced) expect(files, file).toContain(file);
  });
});
