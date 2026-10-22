import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";
import { ResumeBuilderEditorPage } from "../pages/ResumeBuilderEditorPage";

test.describe("Resume Builder — Auto-save", () => {
  test("persists edited summary content after a full reload", async ({
    page,
  }) => {
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.createResume();
    const editor = new ResumeBuilderEditorPage(page);
    const content = `Autosave regression ${Date.now()}: recent edited summary must survive reload.`;
    await editor.fillSummary(content);
    await editor.waitForSaved();
    await page.reload();
    await expect(
      page.getByRole("textbox", {
        name: "Write a brief summary of your professional background...",
        exact: true,
      }),
    ).toHaveText(content);
  });

  test("persists changed typography after reload", async ({ page }) => {
    const list = new ResumeBuilderListPage(page);
    await list.goto();
    await list.createResume();
    const editor = new ResumeBuilderEditorPage(page);
    await editor.openTypography();
    const fontSize = page.locator('input[type="range"][min="8"]');
    await expect(fontSize).toBeVisible();
    await fontSize.fill("16");
    await editor.waitForSaved();
    await page.reload();
    await editor.openTypography();
    await expect(fontSize).toHaveValue("16");
  });
});
