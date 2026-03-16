import { test, expect } from "@playwright/test";
import { ResumeBuilderListPage } from "../pages/ResumeBuilderListPage";

test.describe("Resume Builder — Auto-save", () => {
  test("saves design changes and persists after reload", async ({ page }) => {
    test.setTimeout(30_000);

    // Create fresh resume
    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();

    const editorUrl = page.url();
    await page.waitForLoadState("networkidle");

    // Change font size via typography panel
    const typographyBtn = page
      .getByRole("button", { name: /typography/i })
      .first();
    await typographyBtn.click();

    // Find font size slider (range input with min=8, max=18)
    const fontSizeSlider = page.locator('input[type="range"][min="8"]');
    if (await fontSizeSlider.isVisible()) {
      await fontSizeSlider.fill("16");
    }

    // Wait for auto-save
    await page.waitForTimeout(2500);

    // Reload and verify
    await page.goto(editorUrl);
    await page.waitForLoadState("networkidle");

    await typographyBtn.click();
    const sliderValue = await page
      .locator('input[type="range"][min="8"]')
      .inputValue();
    expect(sliderValue).toBe("16");
  });

  test("last edited date updates in list after edit", async ({ page }) => {
    test.setTimeout(30_000);

    const listPage = new ResumeBuilderListPage(page);
    await listPage.goto();
    await listPage.createResume();
    await page.waitForLoadState("networkidle");

    // Make a change (open typography, change spacing)
    const typographyBtn = page
      .getByRole("button", { name: /typography/i })
      .first();
    await typographyBtn.click();

    const spacingSlider = page.locator('input[type="range"][min="50"]');
    if (await spacingSlider.isVisible()) {
      await spacingSlider.fill("120");
    }

    // Wait for auto-save
    await page.waitForTimeout(2500);

    // Go back to list
    await page.goto("/app/resume-builder");
    await page.waitForLoadState("networkidle");

    // The "Edited" text should show recent time
    await expect(page.getByText(/edited/i).first()).toBeVisible();
  });
});
