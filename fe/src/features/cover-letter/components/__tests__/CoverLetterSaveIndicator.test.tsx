import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { CoverLetterSaveIndicator } from "../CoverLetterSaveIndicator";

const mockStoreState = vi.hoisted(() => ({
  current: {
    saveStatus: "idle" as "idle" | "saving" | "saved" | "error",
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/coverLetterStore", () => ({
  useCoverLetterStore: (
    selector: (state: Record<string, unknown>) => unknown,
  ) => selector(mockStoreState.current),
}));

describe("CoverLetterSaveIndicator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStoreState.current = { saveStatus: "idle" };
  });

  it("renders nothing when saveStatus is idle", () => {
    const { container } = render(<CoverLetterSaveIndicator />);
    expect(container.innerHTML).toBe("");
  });

  it("renders saving indicator", () => {
    mockStoreState.current = { saveStatus: "saving" };
    render(<CoverLetterSaveIndicator />);
    expect(screen.getByText("coverLetter.saving")).toBeInTheDocument();
  });

  it("renders saved indicator with green color", () => {
    mockStoreState.current = { saveStatus: "saved" };
    render(<CoverLetterSaveIndicator />);
    const indicator = screen.getByText("coverLetter.saved").closest("span");
    expect(indicator).toBeInTheDocument();
    expect(indicator?.className).toContain("text-green-600");
  });

  it("renders error indicator with destructive color", () => {
    mockStoreState.current = { saveStatus: "error" };
    render(<CoverLetterSaveIndicator />);
    const indicator = screen
      .getByText("coverLetter.saveFailed")
      .closest("span");
    expect(indicator).toBeInTheDocument();
    expect(indicator?.className).toContain("text-destructive");
  });
});
