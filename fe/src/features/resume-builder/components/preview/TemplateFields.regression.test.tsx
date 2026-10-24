import { afterEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import "@/shared/lib/i18n";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { TEMPLATE_MAP } from "../../lib/templateRegistry";
import { filledResumeFixture } from "./__tests__/filledResumeFixture";

afterEach(() => {
  cleanup();
  useResumeBuilderStore.setState({ resume: null, isDirty: false });
});

describe("saved fields in actual templates", () => {
  it.each(Object.entries(TEMPLATE_MAP).flatMap(([name, Template]) => [
    { name, Template, editable: false }, { name, Template, editable: true },
  ]))("renders filled fields in $name (editable=$editable)", ({ name, Template, editable }) => {
    useResumeBuilderStore.setState({ resume: { ...filledResumeFixture, template_id: name } });
    const { container } = render(<Template editable={editable} />);
    for (const value of ["QA Office", "3.9", "2024-05-06", "Expert", "Full Professional"]) {
      expect(container).toHaveTextContent(value);
    }
    expect(container).not.toHaveTextContent("full_professional");
  });

  it.each(["Expert", "Architecture specialist"])("preserves imported skill level %s", (level) => {
    for (const Template of Object.values(TEMPLATE_MAP)) {
      useResumeBuilderStore.setState({ resume: {
        ...filledResumeFixture,
        skills: filledResumeFixture.skills.map((skill) => ({ ...skill, level })),
      } });
      const { container, unmount } = render(<Template />);
      expect(container).toHaveTextContent(level);
      unmount();
    }
  });

  it.each(["00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"])("omits empty optional metadata in %s preview", (id) => {
    const Template = TEMPLATE_MAP[id];
    useResumeBuilderStore.setState({ resume: {
      ...filledResumeFixture,
      experiences: filledResumeFixture.experiences.map((entry) => ({ ...entry, location: "" })),
      educations: filledResumeFixture.educations.map((entry) => ({ ...entry, gpa: "" })),
      certifications: filledResumeFixture.certifications.map((entry) => ({ ...entry, issue_date: "" })),
      skills: filledResumeFixture.skills.map((entry) => ({ ...entry, level: "" })),
      languages: filledResumeFixture.languages.map((entry) => ({ ...entry, proficiency: "" })),
    } });
    const { container } = render(<Template />);
    expect(container).not.toHaveTextContent(/GPA:|Issue date|Proficiency|Level|\(\)/);
    expect(screen.getByText("QA TypeScript")).toBeInTheDocument();
    expect(screen.getByText("QA English")).toBeInTheDocument();
  });

  it("shows Modern certification dates using sidebar text color", () => {
    const Template = TEMPLATE_MAP["00000000-0000-0000-0000-000000000002"];
    useResumeBuilderStore.setState({ resume: {
      ...filledResumeFixture, layout_mode: "double-left",
      section_order: filledResumeFixture.section_order.map((section) => section.section_key === "certifications" ? { ...section, column: "sidebar" } : section),
    } });
    render(<Template />);
    const date = screen.getByText("2024-05-06");
    expect(date).toHaveClass("opacity-70");
    expect(date).not.toHaveClass("text-gray-500");
  });

  it("keeps Modern sidebar date inputs readable and Escape restores the saved value", () => {
    const Template = TEMPLATE_MAP["00000000-0000-0000-0000-000000000002"];
    useResumeBuilderStore.setState({ resume: {
      ...filledResumeFixture, layout_mode: "double-left",
      section_order: [{ section_key: "certifications", sort_order: 0, is_visible: true, column: "sidebar" }],
    } });
    render(<Template editable />);
    const displayedDate = screen.getByRole("textbox", { name: "Issue date" });
    expect(displayedDate).toHaveClass("opacity-70");
    expect(displayedDate).not.toHaveClass("text-gray-700");
    fireEvent.mouseDown(displayedDate);
    const input = screen.getByLabelText("Issue date");
    expect(input.tagName).toBe("INPUT");
    expect(input).toHaveClass("text-gray-700");
    fireEvent.change(input, { target: { value: "2025-07-08" } });
    expect(useResumeBuilderStore.getState().resume?.certifications[0].issue_date).toBe("2025-07-08");
    fireEvent.keyDown(input, { key: "Escape" });
    expect(useResumeBuilderStore.getState().resume?.certifications[0].issue_date).toBe("2024-05-06");
    expect(screen.getByText("2024-05-06")).toHaveClass("opacity-70");
    expect(screen.getByText("2024-05-06")).not.toHaveClass("text-gray-700");
  });

  it("maps uppercase Minimal language proficiency in Preview", () => {
    const Template = TEMPLATE_MAP["00000000-0000-0000-0000-000000000003"];
    useResumeBuilderStore.setState({ resume: {
      ...filledResumeFixture,
      languages: filledResumeFixture.languages.map((language) => ({ ...language, proficiency: "FULL_PROFESSIONAL" })),
    } });
    render(<Template />);
    expect(screen.getByText("QA English (Full Professional)")).toBeInTheDocument();
  });

  it.each(["pill", "text-only"] as const)("preserves explicit name-only %s customization", (skill_display) => {
    const Template = TEMPLATE_MAP["00000000-0000-0000-0000-000000000005"];
    useResumeBuilderStore.setState({ resume: { ...filledResumeFixture, skill_display } });
    render(<Template />);
    expect(screen.getByText("QA TypeScript")).toBeInTheDocument();
    expect(screen.queryByText("Expert")).not.toBeInTheDocument();
  });

  it("edits Minimal proficiency using its readable label", () => {
    const Template = TEMPLATE_MAP["00000000-0000-0000-0000-000000000003"];
    useResumeBuilderStore.setState({ resume: { ...filledResumeFixture, template_id: "minimal" } });
    render(<Template editable />);
    fireEvent.click(screen.getByRole("button", { name: "Proficiency" }));
    fireEvent.click(screen.getByRole("button", { name: "Native" }));
    expect(useResumeBuilderStore.getState().resume?.languages[0].proficiency).toBe("native");
    expect(screen.getByText("Native")).toBeInTheDocument();
  });
});
