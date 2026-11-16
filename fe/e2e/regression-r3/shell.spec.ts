import { test, expect } from "@playwright/test";

for (const viewport of [{ width: 390, height: 844 }, { width: 1440, height: 900 }]) {
  test(`R3 auth destinations and feature close context at ${viewport.width}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.addInitScript(() => {
      localStorage.removeItem("jobber-auth");
      localStorage.setItem("language", "en");
      localStorage.setItem("cookie-consent", JSON.stringify({ choice: "essential", version: 1 }));
    });
    await page.goto("http://localhost:3003/login?next=%2Fapp%2Fanalytics");
    let dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: "Forgot Password?", exact: true }).click();
    await expect(page).toHaveURL(/forgot-password\?next=%2Fapp%2Fanalytics/);
    await page.getByRole("button", { name: "Back to login", exact: true }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Register", exact: true }).click();
    await expect(page).toHaveURL(/register\?next=%2Fapp%2Fanalytics/);
    await page.getByRole("dialog").getByRole("button", { name: "Login", exact: true }).click();
    await expect(page).toHaveURL(/login\?next=%2Fapp%2Fanalytics/);
    await page.getByRole("dialog").getByRole("button", { name: "Close", exact: true }).click();
    for (const feature of ["applications", "resume-builder", "cover-letters"]) {
      await page.goto(`http://localhost:3003/features/${feature}`);
      await page.locator("main section").first().getByRole("button").click();
      await expect(page).toHaveURL(/register\?returnTo=/);
      dialog = page.getByRole("dialog");
      await dialog.getByRole("button", { name: "Login", exact: true }).click();
      await page.getByRole("dialog").getByRole("button", { name: "Close", exact: true }).click();
      await expect(page).toHaveURL(`http://localhost:3003/features/${feature}`);
      expect((await page.title()).match(/Cursopus/g)).toHaveLength(1);
    }
  });

  test(`R3 zero funnel, branded title and logout without refresh at ${viewport.width}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    const user = { id: "shell-user", name: "Synthetic", email: "synthetic@example.invalid", locale: "en", created_at: "2026-01-01" };
    await page.addInitScript((user) => {
      localStorage.setItem("language", "en");
      localStorage.setItem("jobber-onboarding-completed", "true");
      localStorage.setItem("cookie-consent", JSON.stringify({ choice: "essential", version: 1 }));
      localStorage.setItem("jobber-auth", JSON.stringify({ state: { user, isAuthenticated: true }, version: 0 }));
    }, user);
    let logout = false;
    const afterLogout: string[] = [];
    await page.route("**/api/v1/**", async (route) => {
      const path = new URL(route.request().url()).pathname.replace("/api/v1/", "");
      if (path === "auth/logout") { logout = true; return route.fulfill({ json: {} }); }
      if (logout) afterLogout.push(path);
      const json = path === "subscription" ? { plan: "free", limits: {}, usage: {} }
        : path === "profile" ? user
        : path === "analytics/overview" ? { total_applications: 100, active_applications: 99, closed_applications: 1, response_rate: 1, avg_days_to_first_response: 2 }
        : path === "analytics/funnel" ? { stages: [
          { stage_name: "applied", stage_order: 0, count: 100, conversion_rate: 100, drop_off_rate: 0 },
          { stage_name: "offer", stage_order: 1, count: 0, conversion_rate: 0, drop_off_rate: 100 },
        ] } : path === "analytics/resumes" ? { resumes: [] } : path === "analytics/sources" ? { sources: [] } : { stages: [] };
      await route.fulfill({ json });
    });
    await page.goto("http://localhost:3003/app/analytics");
    await expect(page).toHaveTitle("Analytics — Cursopus");
    const bars = page.locator('[style*="--bar-w"]');
    await expect(bars).toHaveCount(2);
    await expect(bars.nth(1)).toHaveCSS("width", "0px");
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
    await page.goto("http://localhost:3003/app/settings");
    await expect(page).toHaveTitle("Settings — Cursopus");
    await page.locator("main").getByRole("button", { name: "Logout", exact: true }).click();
    await expect(page).toHaveURL("http://localhost:3003/");
    expect(afterLogout).not.toContain("auth/refresh");
    expect(afterLogout).not.toContain("subscription");
    expect(await page.evaluate(() => localStorage.getItem("jobber-auth"))).toBeNull();
  });
}
