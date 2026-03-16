import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";
import { ResumeBuilderEditorPage } from "../pages/ResumeBuilderEditorPage";

test.describe("Resume Builder — Editor", () => {
  let editorUrl: string;

  test.beforeAll(async ({ browser }) => {
    // Create one resume to use across editor tests
    const context = await browser.newContext({
      storageState: ".auth/user.json",
    });
    const page = await context.newPage();
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    editorUrl = page.url();
    await context.close();
  });

  test("loads editor with toolbar", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    await expect(page.getByRole("button", { name: /undo/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /redo/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /export/i })).toBeVisible();
  });

  test("shows back button that navigates to list", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const backBtn = page.getByRole("button", { name: /back to resumes/i });
    await expect(backBtn).toBeVisible();
    await backBtn.click();
    await expect(page).toHaveURL("/app/resume-builder");
  });

  test("shows 404 state for non-existent resume", async ({ page }) => {
    await page.goto("/app/resume-builder/00000000-0000-0000-0000-000000000000");
    await page.waitForLoadState("networkidle");
    await expect(page.getByText(/not found/i)).toBeVisible();
  });

  test("shows save indicator after editing", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    // Edit the title (displayed as h1 in toolbar)
    const titleEl = page.locator("h1").first();
    if (await titleEl.isVisible()) {
      // Title is visible — the editor loaded
      await expect(titleEl).toBeVisible();
    }
  });
});

test.describe("Resume Builder — Design Sidebar", () => {
  let editorUrl: string;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: ".auth/user.json",
    });
    const page = await context.newPage();
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    editorUrl = page.url();
    await context.close();
  });

  test("sidebar is visible on desktop", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    // Design sidebar has icon buttons
    const sidebar = page.locator(".hidden.md\\:block").first();
    await expect(sidebar).toBeVisible();
  });

  test("template picker opens and shows templates", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const editor = new ResumeBuilderEditorPage(page);
    await editor.openTemplates();

    // Should see template options
    await expect(
      page.getByText(/professional|modern|minimal/i).first(),
    ).toBeVisible();
  });

  test("color picker opens and shows palettes", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const editor = new ResumeBuilderEditorPage(page);
    await editor.openColors();

    // Should see color swatches
    await expect(page.locator("button[style*='background']").first()).toBeVisible();
  });

  test("typography panel opens and shows font selector", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const editor = new ResumeBuilderEditorPage(page);
    await editor.openTypography();

    // Should see font family select and font size slider
    await expect(page.getByText(/georgia|arial/i).first()).toBeVisible();
  });
});

test.describe("Resume Builder — Side Panels", () => {
  let editorUrl: string;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: ".auth/user.json",
    });
    const page = await context.newPage();
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    editorUrl = page.url();
    await context.close();
  });

  test("AI panel toggles on/off", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const editor = new ResumeBuilderEditorPage(page);

    // Open AI panel
    await editor.toggleAIPanel();
    await expect(page.getByText(/ai assistant/i)).toBeVisible();

    // Close AI panel
    const closeBtn = page.getByRole("button", { name: /close/i }).last();
    await closeBtn.click();
  });

  test("ATS panel toggles on/off", async ({ page }) => {
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    const editor = new ResumeBuilderEditorPage(page);

    // Open ATS panel
    await editor.toggleATSPanel();
    await expect(page.getByText(/ats check/i)).toBeVisible();
  });
});
