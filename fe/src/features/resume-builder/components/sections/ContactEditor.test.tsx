import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ContactEditor } from "./ContactEditor";
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

describe("ContactEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<ContactEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.contact"),
    ).toBeInTheDocument();
  });

  it("renders all contact fields with correct values", () => {
    render(<ContactEditor />);
    const fullNameInput = screen.getByLabelText(
      "resumeBuilder.contact.fullName",
    );
    expect(fullNameInput).toHaveValue("John Doe");

    const emailInput = screen.getByLabelText("resumeBuilder.contact.email");
    expect(emailInput).toHaveValue("john@example.com");

    const phoneInput = screen.getByLabelText("resumeBuilder.contact.phone");
    expect(phoneInput).toHaveValue("+1234567890");

    const locationInput = screen.getByLabelText(
      "resumeBuilder.contact.location",
    );
    expect(locationInput).toHaveValue("New York, NY");

    const websiteInput = screen.getByLabelText("resumeBuilder.contact.website");
    expect(websiteInput).toHaveValue("https://johndoe.com");

    const linkedinInput = screen.getByLabelText(
      "resumeBuilder.contact.linkedin",
    );
    expect(linkedinInput).toHaveValue("linkedin.com/in/johndoe");

    const githubInput = screen.getByLabelText("resumeBuilder.contact.github");
    expect(githubInput).toHaveValue("github.com/johndoe");
  });

  it("calls updateContact when a field is changed", () => {
    render(<ContactEditor />);

    const fullNameInput = screen.getByLabelText(
      "resumeBuilder.contact.fullName",
    );
    fireEvent.change(fullNameInput, { target: { value: "Jane Smith" } });

    expect(mockState.updateContact).toHaveBeenCalledTimes(1);
    expect(mockState.updateContact).toHaveBeenCalledWith(
      expect.objectContaining({ full_name: "Jane Smith" }),
    );
  });

  it("renders default empty values when contact is null", () => {
    const nullContactState = createMockStoreState({ contact: null });
    Object.assign(mockState, nullContactState);

    render(<ContactEditor />);
    const fullNameInput = screen.getByLabelText(
      "resumeBuilder.contact.fullName",
    );
    expect(fullNameInput).toHaveValue("");
  });

  it("updates email field correctly", () => {
    render(<ContactEditor />);

    const emailInput = screen.getByLabelText("resumeBuilder.contact.email");
    fireEvent.change(emailInput, { target: { value: "new@email.com" } });

    expect(mockState.updateContact).toHaveBeenCalledTimes(1);
    expect(mockState.updateContact).toHaveBeenCalledWith(
      expect.objectContaining({ email: "new@email.com" }),
    );
  });
});
