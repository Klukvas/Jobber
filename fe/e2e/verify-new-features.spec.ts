import { test, expect } from "@playwright/test";

// E2E verification of the four features added in this branch:
//  1. Landing feature cards use SVG icons (not emoji)
//  2. Company/title search on the applications list
//  3. Tags on the job detail (create + attach)
//  4. Reminders on the job detail (create)
//
// Runs against the live dev stack (reuseExistingServer) using the seed user
// session from global-setup. Screenshots are written to /tmp/jobber-e2e.

const SHOTS = "/tmp/jobber-e2e";

test.describe.configure({ mode: "serial" });

// Auth comes from the shared storageState created by global-setup. A fresh
// context has no localStorage though, so the onboarding wizard would show and
// its full-screen backdrop intercepts clicks — mark it completed up-front.
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("jobber-onboarding-completed", "true");
  });
});

test("landing: feature cards render SVG icons, not emoji", async ({ page }) => {
  await page.goto("/");

  const features = page.locator("#features");
  await features.scrollIntoViewIfNeeded();
  await expect(features).toBeVisible();

  // Every feature card icon is now an inline <svg> (lucide).
  const svgCount = await features.locator("svg").count();
  expect(svgCount).toBeGreaterThanOrEqual(6);

  // The old emoji glyphs must be gone.
  for (const emoji of ["🗂", "🤖", "📄", "🔗", "📊", "🗓"]) {
    await expect(features).not.toContainText(emoji);
  }

  await page.screenshot({
    path: `${SHOTS}/01-landing-features.png`,
    fullPage: true,
  });
});

test("applications: company search filters the list", async ({ page }) => {
  await page.goto("/app/jobs");

  const search = page.getByPlaceholder(/search|поиск|пошук/i);
  await expect(search).toBeVisible();

  // Positive match: a seed company with jobs.
  await search.fill("TechNova");
  await page.waitForTimeout(700); // debounce (300ms) + request
  await expect(page.getByText("TechNova").first()).toBeVisible();
  await page.screenshot({
    path: `${SHOTS}/02a-search-match.png`,
    fullPage: true,
  });

  // Negative match: nothing should remain.
  await search.fill("zzznomatchqqq123");
  await page.waitForTimeout(700);
  await expect(page.getByText("TechNova")).toHaveCount(0);
  await page.screenshot({
    path: `${SHOTS}/02b-search-empty.png`,
    fullPage: true,
  });
});

test("job detail: create a tag and a reminder end-to-end", async ({ page }) => {
  await page.goto("/app/jobs");

  // Open the first job's detail page (navigate by href — deterministic).
  const firstJob = page.locator('a[href*="/app/jobs/"]').first();
  await expect(firstJob).toBeVisible();
  const href = await firstJob.getAttribute("href");
  expect(href).toBeTruthy();
  await page.goto(href!);
  await page.waitForURL(/\/app\/jobs\/[0-9a-fA-F-]{36}/);

  const stamp = Date.now();

  // --- Tags: create a new tag (auto-attaches to this job) ---
  const tagName = `e2e-tag-${stamp}`;
  const tagInput = page.getByPlaceholder(/new tag name|нового тега/i);
  await tagInput.scrollIntoViewIfNeeded();
  await tagInput.fill(tagName);
  const tagForm = page.locator("form", { has: tagInput });
  await tagForm.getByRole("button").click();
  await expect(page.getByText(tagName)).toBeVisible({ timeout: 10_000 });

  // --- Reminders: create a reminder ---
  const reminderMsg = `e2e reminder ${stamp}`;
  const reminderInput = page.getByPlaceholder(/remind me|напомнить|нагадати/i);
  await reminderInput.scrollIntoViewIfNeeded();
  await reminderInput.fill(reminderMsg);
  await page.locator('input[type="datetime-local"]').fill("2026-12-31T10:00");
  const reminderForm = page.locator("form", { has: reminderInput });
  await reminderForm
    .getByRole("button", { name: /reminder|напоминание|нагадування/i })
    .click();
  await expect(page.getByText(reminderMsg)).toBeVisible({ timeout: 10_000 });

  await page.screenshot({
    path: `${SHOTS}/03-job-tags-reminders.png`,
    fullPage: true,
  });
});
