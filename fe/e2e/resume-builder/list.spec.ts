import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";

test.describe("Resume Builder — List Page", () => {
  test("navigates to resume builder list", async ({ page }) => {
    await page.goto("/app/resume-builder");
    await expect(page).toHaveURL("/app/resume-builder");
    await expect(
      page.getByRole("heading", { name: /resume builder/i }),
    ).toBeVisible();
  });

  test("shows create button", async ({ page }) => {
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await expect(listPage.createButton).toBeVisible();
  });

  test("creates a new resume and redirects to editor", async ({ page }) => {
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    await expect(page).toHaveURL(/\/app\/resume-builder\/.+/);
  });

  test("shows import button", async ({ page }) => {
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await expect(listPage.importButton).toBeVisible();
  });
});

test.describe("Resume Builder — CRUD", () => {
  let resumeId: string;

  test.beforeEach(async ({ page }) => {
    // Create a fresh resume for each test
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    // Extract ID from URL
    const url = page.url();
    resumeId = url.split("/").pop()!;
  });

  test("new resume appears in list with 'Untitled Resume' title", async ({
    page,
  }) => {
    await page.goto("/app/resume-builder");
    await page.waitForLoadState("networkidle");
    await expect(page.getByText("Untitled Resume").first()).toBeVisible();
  });

  test("duplicate creates a copy with (Copy) suffix", async ({ page }) => {
    await page.goto("/app/resume-builder");
    await page.waitForLoadState("networkidle");

    const countBefore = await new ResumeBuilderListPage(page).getResumeCount();

    // Find and duplicate
    const duplicateBtn = page.getByRole("button", { name: /duplicate/i }).first();
    await duplicateBtn.click();
    await page.waitForLoadState("networkidle");

    const countAfter = await new ResumeBuilderListPage(page).getResumeCount();
    expect(countAfter).toBe(countBefore + 1);
    await expect(page.getByText(/\(Copy\)/).first()).toBeVisible();
  });

  test("delete removes resume from list", async ({ page }) => {
    await page.goto("/app/resume-builder");
    await page.waitForLoadState("networkidle");

    const countBefore = await new ResumeBuilderListPage(page).getResumeCount();

    // Delete first resume
    const deleteBtn = page.getByRole("button", { name: /delete/i }).first();
    await deleteBtn.click();

    // Confirm in dialog
    const confirmBtn = page
      .getByRole("alertdialog")
      .getByRole("button", { name: /delete/i });
    await confirmBtn.click();

    await page.waitForLoadState("networkidle");
    const countAfter = await new ResumeBuilderListPage(page).getResumeCount();
    expect(countAfter).toBe(countBefore - 1);
  });
});
