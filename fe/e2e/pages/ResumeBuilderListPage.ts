import { type Page, type Locator } from "@playwright/test";

export class ResumeBuilderListPage {
  readonly heading: Locator;
  readonly createButton: Locator;
  readonly importButton: Locator;
  readonly resumeCards: Locator;
  readonly emptyState: Locator;

  constructor(private readonly page: Page) {
    this.heading = page.getByRole("heading", { name: /resume builder/i });
    this.createButton = page.getByRole("button", { name: /new resume/i });
    this.importButton = page.getByRole("button", { name: /import/i });
    this.resumeCards = page.locator("[class*='grid'] > div").filter({ has: page.locator("img, svg, canvas") });
    this.emptyState = page.getByText(/no resumes yet/i);
  }

  async goto() {
    await this.page.goto("/app/resume-builder");
    await this.page.waitForLoadState("networkidle");
  }

  async createResume() {
    await this.createButton.click();
    await this.page.waitForURL("/app/resume-builder/*", { timeout: 10_000 });
  }

  async getResumeCount(): Promise<number> {
    // Each resume is a card/link in the grid
    const links = this.page.locator('a[href^="/app/resume-builder/"]');
    return links.count();
  }

  async openFirstResume() {
    const link = this.page.locator('a[href^="/app/resume-builder/"]').first();
    await link.click();
    await this.page.waitForURL("/app/resume-builder/*", { timeout: 10_000 });
  }

  async duplicateResume(index = 0) {
    const cards = this.page.locator('a[href^="/app/resume-builder/"]');
    const card = cards.nth(index);
    // Duplicate button is inside the card actions
    const duplicateBtn = card.getByRole("button", { name: /duplicate/i });
    await duplicateBtn.click();
  }

  async deleteResume(index = 0) {
    const cards = this.page.locator('a[href^="/app/resume-builder/"]');
    const card = cards.nth(index);
    const deleteBtn = card.getByRole("button", { name: /delete/i });
    await deleteBtn.click();
    // Confirm dialog
    const confirmBtn = this.page.getByRole("button", { name: /delete|confirm/i }).last();
    await confirmBtn.click();
  }
}
