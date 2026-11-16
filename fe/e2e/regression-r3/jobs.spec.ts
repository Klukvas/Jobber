import { test, expect } from "@playwright/test";

for (const viewport of [{ width: 390, height: 844 }, { width: 1440, height: 900 }]) {
  test(`R3 list keyboard actions, usage 100→99 and HTTP(S) validation at ${viewport.width}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.addInitScript(() => {
      localStorage.setItem("language", "en");
      localStorage.setItem("jobber-onboarding-completed", "true");
      localStorage.setItem("cookie-consent", JSON.stringify({ choice: "essential", version: 1 }));
      localStorage.setItem("jobber-auth", JSON.stringify({ state: {
        user: { id: "r3-user", name: "Synthetic Candidate", email: "synthetic@example.invalid", locale: "en", created_at: "2026-01-01" }, isAuthenticated: true,
      }, version: 0 }));
    });
    let deleted = false;
    let deletes = 0;
    const timestamp = "2026-01-01T00:00:00Z";
    const jobs = ["One", "Two"].map((title, index) => ({ id: `job-${index}`, title, is_favorite: false, is_archived: false, created_at: timestamp, updated_at: timestamp, last_activity_at: timestamp }));
    await page.route("**/*", async (route) => {
      const url = new URL(route.request().url());
      if (!["localhost", "127.0.0.1"].includes(url.hostname)) return route.abort();
      if (!url.pathname.startsWith("/api/v1/")) return route.continue();
      const path = url.pathname.replace("/api/v1/", "");
      if (path === "jobs/job-0" && route.request().method() === "DELETE") {
        deletes++;
        await new Promise((resolve) => setTimeout(resolve, 200));
        deleted = true;
        return route.fulfill({ status: 204 });
      }
      const json = path === "subscription" ? {
        plan: "pro", status: "active", limits: { max_jobs: 100, max_resumes: 10, max_ai_requests: 500, max_job_parses: 0, max_resume_builders: 10, max_cover_letters: 10 },
        usage: { jobs: deleted ? 99 : 100, resumes: 0, ai_requests: 0, job_parses: 0, resume_builders: 0, cover_letters: 0 },
      } : path === "jobs" ? { items: deleted ? jobs.slice(1) : jobs, total: deleted ? 99 : 100, limit: 20, offset: 0 }
        : path === "resume-builder" ? [] : { items: [], total: 0 };
      return route.fulfill({ json });
    });
    await page.goto("http://localhost:3003/app/jobs?view=list");
    await expect(page.getByRole("link", { name: /One/ })).toBeVisible();
    const create = page.getByRole("button", { name: "Create Job", exact: true });
    await create.click();
    let dialog = page.getByRole("dialog");
    await dialog.getByLabel(/Job Title/).fill("Synthetic job");
    await expect(dialog.getByRole("button", { name: "Create", exact: true })).toBeDisabled();
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();

    const actions = page.getByRole("button", { name: "Job actions", exact: true }).first();
    await actions.focus();
    await actions.press("ArrowDown");
    await expect(page.getByRole("menuitem", { name: "Add Comment" })).toBeFocused();
    await page.keyboard.press("End");
    await expect(page.getByRole("menuitem", { name: "Delete Job" })).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(actions).toBeFocused();
    await expect(page.getByRole("menu")).toHaveCount(0);
    await actions.press("ArrowDown");
    await page.keyboard.press("End");
    await page.keyboard.press("Enter");
    dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "Delete", exact: true }).dblclick();
    await expect(dialog).toHaveCount(0);
    expect(deletes).toBe(1);
    await create.click();
    dialog = page.getByRole("dialog");
    await dialog.getByLabel(/Job Title/).fill("Synthetic job");
    await expect(dialog.getByRole("button", { name: "Create", exact: true })).toBeEnabled();
    await dialog.getByLabel(/^URL/).fill("ftp://example.invalid/job");
    await expect(dialog.getByLabel(/^URL/)).toHaveAttribute("aria-invalid", "true");
    await expect(dialog.getByRole("alert")).toContainText("http://");
    await expect(dialog.getByRole("button", { name: "Create", exact: true })).toBeDisabled();
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
  });
}
