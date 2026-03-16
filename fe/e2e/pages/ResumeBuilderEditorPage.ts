import { type Page, type Locator, expect } from "@playwright/test";

export class ResumeBuilderEditorPage {
  readonly backButton: Locator;
  readonly title: Locator;
  readonly saveIndicator: Locator;
  readonly aiButton: Locator;
  readonly atsButton: Locator;
  readonly exportButton: Locator;
  readonly undoButton: Locator;
  readonly redoButton: Locator;
  readonly preview: Locator;

  constructor(private readonly page: Page) {
    this.backButton = page.getByRole("button", { name: /back to resumes/i });
    this.title = page.locator("h1");
    this.saveIndicator = page.getByText(/saved|saving/i);
    this.aiButton = page.getByRole("button", { name: /ai/i });
    this.atsButton = page.getByRole("button", { name: /ats/i });
    this.exportButton = page.getByRole("button", { name: /export/i });
    this.undoButton = page.getByRole("button", { name: /undo/i });
    this.redoButton = page.getByRole("button", { name: /redo/i });
    this.preview = page.locator("[class*='bg-muted']").first();
  }

  async goto(id: string) {
    await this.page.goto(`/app/resume-builder/${id}`);
    await this.page.waitForLoadState("networkidle");
  }

  async waitForSaved() {
    // Wait for save indicator to show "Saved"
    await expect(this.page.getByText(/saved/i).first()).toBeVisible({
      timeout: 5_000,
    });
  }

  // --- Sidebar design ---

  async openTemplates() {
    await this.page
      .getByRole("button", { name: /templates/i })
      .first()
      .click();
  }

  async openColors() {
    await this.page.getByRole("button", { name: /colors/i }).first().click();
  }

  async openTypography() {
    await this.page
      .getByRole("button", { name: /typography/i })
      .first()
      .click();
  }

  async openLayout() {
    await this.page.getByRole("button", { name: /layout/i }).first().click();
  }

  // --- Section editing ---

  async fillContactField(label: string, value: string) {
    const input = this.page.getByLabel(label, { exact: false });
    await input.clear();
    await input.fill(value);
    // Tab away to trigger blur/save
    await input.press("Tab");
  }

  async fillSummary(text: string) {
    const textarea = this.page
      .getByLabel(/summary/i)
      .or(this.page.locator("textarea").first());
    await textarea.clear();
    await textarea.fill(text);
    await textarea.press("Tab");
  }

  async addExperience() {
    // Navigate to experience section first
    const addBtn = this.page
      .getByRole("button", { name: /add/i })
      .filter({ hasText: /add/i })
      .first();
    await addBtn.click();
  }

  // --- Side panels ---

  async toggleAIPanel() {
    await this.aiButton.click();
  }

  async toggleATSPanel() {
    await this.atsButton.click();
  }

  // --- Export ---

  async exportPDF() {
    await this.exportButton.click();
    const pdfBtn = this.page.getByRole("menuitem", { name: /pdf/i });
    await pdfBtn.click();
  }
}
