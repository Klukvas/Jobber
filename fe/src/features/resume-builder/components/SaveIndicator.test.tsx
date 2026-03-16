import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { SaveIndicator } from "./SaveIndicator";
import { createMockStoreState } from "./__tests__/testHelpers";

const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

describe("SaveIndicator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when saveStatus is idle", () => {
    const { container } = render(<SaveIndicator />);
    expect(container.innerHTML).toBe("");
  });

  it("renders saving indicator", () => {
    Object.assign(mockState, { saveStatus: "saving" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saving")).toBeInTheDocument();
  });

  it("renders saved indicator", () => {
    Object.assign(mockState, { saveStatus: "saved" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saved")).toBeInTheDocument();
  });

  it("renders error indicator", () => {
    Object.assign(mockState, { saveStatus: "error" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saveFailed")).toBeInTheDocument();
  });

  it("applies green color for saved status", () => {
    Object.assign(mockState, { saveStatus: "saved" });
    render(<SaveIndicator />);
    const indicator = screen.getByText("resumeBuilder.saved").closest("span");
    expect(indicator?.className).toContain("text-green-600");
  });

  it("applies destructive color for error status", () => {
    Object.assign(mockState, { saveStatus: "error" });
    render(<SaveIndicator />);
    const indicator = screen
      .getByText("resumeBuilder.saveFailed")
      .closest("span");
    expect(indicator?.className).toContain("text-destructive");
  });
});
