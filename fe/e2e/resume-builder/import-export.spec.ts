import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";

test.describe("Resume Builder — Import", () => {
  test("import modal opens with text and PDF tabs", async ({ page }) => {
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();

    await listPage.importButton.click();

    // Modal should be visible
    await expect(page.getByText(/import resume/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /paste text/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /upload pdf/i })).toBeVisible();
  });

  test("import from text creates a new resume", async ({ page }) => {
    test.setTimeout(30_000);

    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();

    const countBefore = await listPage.getResumeCount();

    await listPage.importButton.click();

    // Fill in resume text
    const textarea = page.getByPlaceholder(/paste your resume/i);
    await textarea.fill(
      "John Doe\nSoftware Engineer\njohn@example.com\n\nExperience:\nSenior Developer at TechCorp (2020-2024)\n- Built scalable microservices\n- Led team of 5 engineers\n\nEducation:\nBS Computer Science, MIT (2016-2020)",
    );

    // Click import
    const importBtn = page.getByRole("button", { name: /^import$/i }).last();
    await importBtn.click();

    // Wait for redirect to editor (import creates resume and redirects)
    await page.waitForURL(/\/app\/resume-builder\/.+/, { timeout: 20_000 });

    // Verify resume was created by going back to list
    await page.goto("/app/resume-builder");
    await page.waitForLoadState("networkidle");

    const countAfter = await listPage.getResumeCount();
    expect(countAfter).toBeGreaterThan(countBefore);
  });
});

test.describe("Resume Builder — Export", () => {
  test("export dropdown shows PDF and DOCX options", async ({ page }) => {
    test.setTimeout(30_000);

    // Create a resume first
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    await page.waitForLoadState("networkidle");

    // Click export button
    const exportBtn = page.getByRole("button", { name: /export/i });
    await exportBtn.click();

    // Should see PDF and DOCX options
    await expect(page.getByText(/pdf/i)).toBeVisible();
    await expect(page.getByText(/docx/i)).toBeVisible();
  });

  test("export PDF triggers download", async ({ page }) => {
    test.setTimeout(60_000);

    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    await page.waitForLoadState("networkidle");

    // Start waiting for download before clicking
    const downloadPromise = page.waitForEvent("download", { timeout: 60_000 });

    const exportBtn = page.getByRole("button", { name: /export/i });
    await exportBtn.click();

    const pdfOption = page.getByRole("menuitem", { name: /pdf/i }).or(
      page.getByText(/export pdf/i),
    );
    await pdfOption.click();

    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
  });
});
