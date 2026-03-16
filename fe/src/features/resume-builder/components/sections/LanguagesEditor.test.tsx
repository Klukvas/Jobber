import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LanguagesEditor } from "./LanguagesEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { LanguageDTO } from "@/shared/types/resume-builder";

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

const sampleLanguage: LanguageDTO = {
  id: "lang-1",
  name: "English",
  proficiency: "native",
  sort_order: 0,
};

describe("LanguagesEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<LanguagesEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.languages"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no languages", () => {
    render(<LanguagesEditor />);
    expect(
      screen.getByText("resumeBuilder.languages.empty"),
    ).toBeInTheDocument();
  });

  it("adds language when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<LanguagesEditor />);

    await user.click(screen.getByText("resumeBuilder.actions.add"));
    expect(mockState.addLanguage).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders language entry with name and proficiency", () => {
    const stateWithLangs = createMockStoreState({
      languages: [sampleLanguage],
    });
    Object.assign(mockState, stateWithLangs);

    render(<LanguagesEditor />);
    expect(screen.getByDisplayValue("English")).toBeInTheDocument();
    // The select displays the i18n key for the selected option
    const profSelect = screen.getByDisplayValue(
      "resumeBuilder.languages.proficiencies.native",
    );
    expect(profSelect).toBeInTheDocument();
  });

  it("calls updateLanguage when name changes", async () => {
    const user = userEvent.setup();
    const stateWithLangs = createMockStoreState({
      languages: [sampleLanguage],
    });
    Object.assign(mockState, stateWithLangs);

    render(<LanguagesEditor />);
    const nameInput = screen.getByDisplayValue("English");
    await user.clear(nameInput);
    await user.type(nameInput, "Spanish");

    expect(mockState.updateLanguage).toHaveBeenCalled();
  });

  it("calls removeLanguage when remove button is clicked", async () => {
    const user = userEvent.setup();
    const stateWithLangs = createMockStoreState({
      languages: [sampleLanguage],
    });
    Object.assign(mockState, stateWithLangs);

    render(<LanguagesEditor />);
    const removeBtn = screen.getByLabelText("resumeBuilder.actions.remove");
    await user.click(removeBtn);

    expect(mockState.removeLanguage).toHaveBeenCalledWith("lang-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("lang-1");
  });

  it("does not show empty state when languages exist", () => {
    const stateWithLangs = createMockStoreState({
      languages: [sampleLanguage],
    });
    Object.assign(mockState, stateWithLangs);

    render(<LanguagesEditor />);
    expect(
      screen.queryByText("resumeBuilder.languages.empty"),
    ).not.toBeInTheDocument();
  });
});
