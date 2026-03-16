import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CustomSectionsEditor } from "./CustomSectionsEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { CustomSectionDTO } from "@/shared/types/resume-builder";

const mockState = createMockStoreState();
const mockPersistAdd = vi.fn();
const mockPersistRemove = vi.fn();

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

vi.mock("../../hooks/useSectionPersistence", () => ({
  useSectionPersistence: () => ({
    add: mockPersistAdd,
    remove: mockPersistRemove,
  }),
}));

const sampleCustomSection: CustomSectionDTO = {
  id: "cs-1",
  title: "Publications",
  content: "Published paper on AI in IEEE",
  sort_order: 0,
};

describe("CustomSectionsEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<CustomSectionsEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.customSections"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no custom sections", () => {
    render(<CustomSectionsEditor />);
    expect(
      screen.getByText("resumeBuilder.customSections.empty"),
    ).toBeInTheDocument();
  });

  it("adds a custom section when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<CustomSectionsEditor />);

    await user.click(screen.getByText("resumeBuilder.customSections.add"));
    expect(mockState.addCustomSection).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders custom section card with title", () => {
    const stateWithCustom = createMockStoreState({
      custom_sections: [sampleCustomSection],
    });
    Object.assign(mockState, stateWithCustom);

    render(<CustomSectionsEditor />);
    expect(screen.getByText("Publications")).toBeInTheDocument();
  });

  it("displays title and content fields when expanded", () => {
    const stateWithCustom = createMockStoreState({
      custom_sections: [sampleCustomSection],
    });
    Object.assign(mockState, stateWithCustom);

    render(<CustomSectionsEditor />);
    // Cards start open by default
    expect(screen.getByDisplayValue("Publications")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("Published paper on AI in IEEE"),
    ).toBeInTheDocument();
  });

  it("calls removeCustomSection when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithCustom = createMockStoreState({
      custom_sections: [sampleCustomSection],
    });
    Object.assign(mockState, stateWithCustom);

    render(<CustomSectionsEditor />);
    await user.click(screen.getByText("resumeBuilder.actions.remove"));

    expect(mockState.removeCustomSection).toHaveBeenCalledWith("cs-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("cs-1");
  });

  it("shows fallback title for empty custom section", () => {
    const emptySection: CustomSectionDTO = {
      ...sampleCustomSection,
      title: "",
    };
    const stateWithEmpty = createMockStoreState({
      custom_sections: [emptySection],
    });
    Object.assign(mockState, stateWithEmpty);

    render(<CustomSectionsEditor />);
    expect(
      screen.getByText("resumeBuilder.customSections.newEntry"),
    ).toBeInTheDocument();
  });

  it("toggles card open/closed", async () => {
    const user = userEvent.setup();
    const stateWithCustom = createMockStoreState({
      custom_sections: [sampleCustomSection],
    });
    Object.assign(mockState, stateWithCustom);

    render(<CustomSectionsEditor />);
    await user.click(screen.getByText("Publications"));

    expect(
      screen.queryByDisplayValue("Publications"),
    ).not.toBeInTheDocument();
  });
});
