import { execFileSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";
import { ResumeBuilderEditorPage } from "../pages/ResumeBuilderEditorPage";

test.describe("Resume Builder — Import", () => {
  test("opens text/PDF import and disables empty input", async ({ page }) => {
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.openImport();
    const dialog = page.getByRole("dialog", {
      name: "Import Resume",
      exact: true,
    });
    await expect(dialog).toBeVisible();
    await expect(
      dialog.getByRole("button", { name: /^import$/i }),
    ).toBeDisabled();
    await dialog.getByRole("button", { name: /^upload pdf$/i }).click();
    await expect(
      dialog.getByRole("button", { name: /^import$/i }),
    ).toBeDisabled();
    await expect(dialog.getByText("PDF up to 5MB")).toBeVisible();
  });

  test("keeps input and explains an unavailable AI import", async ({
    page,
  }) => {
    await page.route("**/resume-builder/import/text", (route) =>
      route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({
          error_code: "AI_UNAVAILABLE",
          error_message: "",
        }),
      }),
    );
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.openImport();
    const text = "Synthetic engineer with TypeScript experience.";
    await page.getByLabel("Resume Text", { exact: true }).fill(text);
    await page.getByRole("button", { name: /^import$/i }).click();
    await expect(
      page.getByText(/AI resume import is unavailable right now/),
    ).toBeVisible();
    await expect(page.getByLabel("Resume Text", { exact: true })).toHaveValue(
      text,
    );
  });
});

test.describe("Resume Builder — Export", () => {
  test("offers PDF and DOCX", async ({ page }) => {
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.createResume();
    await page.getByRole("button", { name: /^export$/i }).click();
    await expect(
      page.getByRole("button", { name: /^export pdf$/i }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: /^export docx$/i }),
    ).toBeVisible();
  });

  test("PDF contains the latest edited text and no editor/privacy placeholders", async ({
    page,
  }) => {
    test.setTimeout(60_000);
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.createResume();
    const editor = new ResumeBuilderEditorPage(page);
    const marker = `LatestExportMarker${Date.now()}`;
    await editor.fillSummary(marker);
    const downloadPromise = page.waitForEvent("download", { timeout: 60_000 });
    await editor.exportPDF();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
    const path = await download.path();
    if (!path) throw new Error("PDF download has no local file");
    expect((await readFile(path)).subarray(0, 5).toString()).toBe("%PDF-");
    // This content regression requires Poppler's pdftotext on PATH. Missing
    // prerequisites fail explicitly instead of silently skipping assertions.
    const text = execFileSync("pdftotext", [path, "-"], { encoding: "utf8" });
    expect(text).toContain(marker);
    expect(text).not.toMatch(
      /We use essential cookies|Essential only|Accept all/,
    );
    expect(text).not.toMatch(
      /Add dates|Write a brief summary|email@example\.com/,
    );
  });
});
