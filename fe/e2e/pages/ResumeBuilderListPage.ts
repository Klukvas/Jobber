import { type Page, type Locator } from "@playwright/test";

export class ResumeBuilderListPage {
  readonly heading: Locator;
  readonly createButton: Locator;
  readonly importButton: Locator;
  readonly resumeCards: Locator;
  readonly emptyState: Locator;

  constructor(private readonly page: Page) {
    this.heading = page.getByRole("heading", { name: /^resumes$/i });
    this.createButton = page.getByRole("button", { name: /^create resume$/i });
    this.importButton = page.getByRole("button", { name: /import/i });
    this.resumeCards = page
      .locator(".group.cursor-pointer")
      .filter({ has: page.getByRole("button", { name: /^duplicate$/i }) });
    this.emptyState = page.getByText(/no resumes yet/i);
  }

  async goto() {
    await this.page.goto("/app/resumes");
    await this.page.waitForLoadState("networkidle");
  }

  async createResume() {
    await this.createButton.click();
    await this.page.getByRole("button", { name: /^build resume$/i }).click();
    await this.page.waitForURL("/app/resume-builder/*", { timeout: 10_000 });
  }

  async openImport() {
    await this.createButton.click();
    await this.importButton.click();
  }

  async getResumeCount(): Promise<number> {
    return this.resumeCards.count();
  }

  async openFirstResume() {
    await this.resumeCards.first().getByRole("heading", { level: 3 }).click();
    await this.page.waitForURL("/app/resume-builder/*", { timeout: 10_000 });
  }

  async duplicateResume(index = 0) {
    const cards = this.resumeCards;
    const card = cards.nth(index);
    // Duplicate button is inside the card actions
    const duplicateBtn = card.getByRole("button", { name: /duplicate/i });
    await duplicateBtn.click();
  }

  async deleteResume(index = 0) {
    const cards = this.resumeCards;
    const card = cards.nth(index);
    const deleteBtn = card.getByRole("button", { name: /delete/i });
    await deleteBtn.click();
    // Confirm dialog
    const confirmBtn = this.page
      .getByRole("dialog")
      .getByRole("button", { name: /^delete$/i });
    await confirmBtn.click();
  }
}
