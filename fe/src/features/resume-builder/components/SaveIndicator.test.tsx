import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

describe("SaveIndicator — unsaved changes", () => {
  beforeEach(() => {
    Object.assign(mockState, createMockStoreState());
  });

  it("shows an unsaved marker as soon as the document is dirty", () => {
    Object.assign(mockState, { isDirty: true, saveStatus: "idle" });
    render(<SaveIndicator />);
    expect(
      screen.getByText("resumeBuilder.unsavedChanges"),
    ).toBeInTheDocument();
  });

  // The window between a keystroke and the debounced save is exactly where a
  // reload used to lose work while the indicator still read "Saved".
  it("replaces a stale 'saved' with the unsaved marker", () => {
    Object.assign(mockState, { isDirty: true, saveStatus: "saved" });
    render(<SaveIndicator />);
    expect(
      screen.getByText("resumeBuilder.unsavedChanges"),
    ).toBeInTheDocument();
    expect(screen.queryByText("resumeBuilder.saved")).not.toBeInTheDocument();
  });

  it("lets an in-flight save show through", () => {
    Object.assign(mockState, { isDirty: true, saveStatus: "saving" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saving")).toBeInTheDocument();
  });

  it("lets a failure show through", () => {
    Object.assign(mockState, { isDirty: true, saveStatus: "error" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saveFailed")).toBeInTheDocument();
  });

  it("shows the saved state once the document is clean", () => {
    Object.assign(mockState, { isDirty: false, saveStatus: "saved" });
    render(<SaveIndicator />);
    expect(screen.getByText("resumeBuilder.saved")).toBeInTheDocument();
  });
});

// A failed save is the one state the user cannot leave on their own — the
// editor has to offer the way back.
describe("SaveIndicator — retry", () => {
  beforeEach(() => {
    Object.assign(mockState, createMockStoreState(), { saveStatus: "error" });
  });

  it("turns the failure into a retry button", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<SaveIndicator onRetry={onRetry} />);

    await user.click(
      screen.getByRole("button", { name: "resumeBuilder.saveFailed" }),
    );

    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("stays a plain message when no retry is offered", () => {
    render(<SaveIndicator />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "resumeBuilder.saveFailed",
    );
  });
});
