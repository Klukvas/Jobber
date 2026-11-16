import { beforeEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CoverLetterPreview } from "../CoverLetterPreview";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  getI18n: () => ({ language: "en" }),
}));

const letter: CoverLetterDTO = {
  id: "editing", resume_builder_id: null, job_id: null, title: "Letter",
  template: "professional", recipient_name: "", recipient_title: "",
  company_name: "", company_address: "", greeting: "Hello",
  paragraphs: ["First paragraph", "Second paragraph"], closing: "Regards",
  font_family: "Georgia", font_size: 12, primary_color: "#2563eb",
  created_at: "", updated_at: "",
};

beforeEach(() => {
  globalThis.ResizeObserver = class {
    observe() {} unobserve() {} disconnect() {}
  } as unknown as typeof ResizeObserver;
  useCoverLetterStore.getState().setCoverLetter(letter);
  useCoverLetterStore.temporal.getState().clear();
});

describe("cover letter paragraph identity", () => {
  it("keeps focus and every character while typing English and Cyrillic", async () => {
    const user = userEvent.setup();
    render(<CoverLetterPreview editable />);
    await user.click(screen.getByText("Second paragraph"));
    const input = screen.getByDisplayValue("Second paragraph");
    await user.type(input, " текст and more", { initialSelectionStart: 16, initialSelectionEnd: 16 });
    expect(input).toHaveFocus();
    expect(input).toHaveValue("Second paragraph текст and more");
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs[1]).toBe("Second paragraph текст and more");
  });

  it("retains the active paragraph when an earlier one is removed and undo restores content", async () => {
    const user = userEvent.setup();
    render(<CoverLetterPreview editable />);
    await user.click(screen.getByText("Second paragraph"));
    const input = screen.getByDisplayValue("Second paragraph");
    act(() => useCoverLetterStore.getState().removeParagraph(0));
    expect(input).toHaveFocus();
    await user.type(input, "!", { initialSelectionStart: 16, initialSelectionEnd: 16 });
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs).toEqual(["Second paragraph!"]);
    act(() => useCoverLetterStore.temporal.getState().undo());
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs).toEqual(["Second paragraph"]);
    act(() => useCoverLetterStore.temporal.getState().undo());
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs).toEqual(letter.paragraphs);
  });
});
