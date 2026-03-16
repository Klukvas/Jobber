import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SummaryEditor } from "./SummaryEditor";
import { createMockStoreState } from "../__tests__/testHelpers";

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

describe("SummaryEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<SummaryEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.summary"),
    ).toBeInTheDocument();
  });

  it("renders the textarea with the current summary content", () => {
    render(<SummaryEditor />);
    const textarea = screen.getByLabelText("resumeBuilder.summary.label");
    expect(textarea).toHaveValue("Experienced software engineer");
  });

  it("displays the character count", () => {
    render(<SummaryEditor />);
    expect(screen.getByText("29/600")).toBeInTheDocument();
  });

  it("calls updateSummary when text changes", () => {
    render(<SummaryEditor />);

    const textarea = screen.getByLabelText("resumeBuilder.summary.label");
    fireEvent.change(textarea, { target: { value: "New summary" } });

    expect(mockState.updateSummary).toHaveBeenCalledTimes(1);
    expect(mockState.updateSummary).toHaveBeenCalledWith({
      content: "New summary",
    });
  });

  it("renders empty textarea when summary is null", () => {
    const nullSummaryState = createMockStoreState({ summary: null });
    Object.assign(mockState, nullSummaryState);

    render(<SummaryEditor />);
    const textarea = screen.getByLabelText("resumeBuilder.summary.label");
    expect(textarea).toHaveValue("");
    expect(screen.getByText("0/600")).toBeInTheDocument();
  });

  it("has maxLength attribute on the textarea", () => {
    render(<SummaryEditor />);
    const textarea = screen.getByLabelText("resumeBuilder.summary.label");
    expect(textarea).toHaveAttribute("maxLength", "600");
  });
});
