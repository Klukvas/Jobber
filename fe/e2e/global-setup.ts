import { test as setup } from "@playwright/test";
import { config } from "dotenv";
import { fileURLToPath } from "url";
import path from "path";
import { STORAGE_STATE } from "./constants";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
config({ path: path.resolve(__dirname, "../.env.test") });

const EMAIL = process.env.PLAYWRIGHT_EMAIL ?? "seed@jobber.dev";
const PASSWORD = process.env.PLAYWRIGHT_PASSWORD ?? "password123";

setup("authenticate", async ({ page }) => {
  await page.goto("/login");

  await page.locator("#login-email").fill(EMAIL);
  await page.locator("#login-password").fill(PASSWORD);
  await page.getByRole("button", { name: /log\s*in|sign\s*in|вход/i }).click();

  // Wait for redirect to app after successful login
  await page.waitForURL("/app/**", { timeout: 15_000 });

  await page.context().storageState({ path: STORAGE_STATE });
});
