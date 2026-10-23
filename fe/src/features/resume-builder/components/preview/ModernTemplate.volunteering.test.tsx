import { afterEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import "@/shared/lib/i18n";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import { ModernTemplate } from "./ModernTemplate";
import { MobileTextEditingContext } from "./mobileTextEditing";

const savedResume: FullResumeDTO = {
  id: "saved-resume",
  title: "Volunteering regression",
  template_id: "modern",
  font_family: "Inter",
  primary_color: "#1e88e5",
  text_color: "#1e88e5",
  spacing: 1,
  margin_top: 20,
  margin_bottom: 20,
  margin_left: 20,
  margin_right: 20,
  layout_mode: "single",
  sidebar_width: 35,
  font_size: 12,
  skill_display: "",
  created_at: "2026-10-23",
  updated_at: "2026-10-23",
  contact: null,
  summary: null,
  experiences: [],
  educations: [],
  skills: [],
  languages: [],
  certifications: [],
  projects: [],
  volunteering: [{
    id: "saved-volunteering",
    role: "QA Volunteer",
    organization: "Synthetic Organization",
    start_date: "",
    end_date: "",
    description: "Synthetic volunteering description\nSecond saved volunteering line",
    sort_order: 0,
  }],
  custom_sections: [],
  section_order: [{
    section_key: "volunteering",
    sort_order: 0,
    is_visible: true,
    column: "main",
  }],
};

afterEach(() => {
  cleanup();
  useResumeBuilderStore.setState({ resume: null, isDirty: false });
});

describe("Modern volunteering preview", () => {
  it.each([
    ["single", "main"],
    ["double-left", "main"],
    ["double-left", "sidebar"],
    ["double-right", "main"],
    ["double-right", "sidebar"],
    ["custom", "sidebar"],
  ] as const)("renders the saved multiline description in %s / %s", (layout, column) => {
    useResumeBuilderStore.setState({
      resume: {
        ...savedResume,
        layout_mode: layout,
        section_order: savedResume.section_order.map((section) => ({ ...section, column })),
      },
      isDirty: false,
    });

    render(<ModernTemplate />);

    expect(screen.getByText("QA Volunteer")).toBeInTheDocument();
    expect(screen.getByText("Synthetic Organization")).toBeInTheDocument();
    const description = screen.getByText(/Synthetic volunteering description/);
    expect(description.textContent).toBe(savedResume.volunteering[0].description);
    expect(description).toHaveClass("whitespace-pre-line");
    if (column === "sidebar") {
      expect(description).not.toHaveClass("text-gray-700");
      expect(description.closest('[style*="background-color"]')).toHaveStyle({
        color: "rgb(255, 255, 255)",
      });
    } else {
      expect(description).toHaveClass("text-gray-700");
    }
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it.each(["", "   "])("omits an empty optional description (%j) in preview", (description) => {
    useResumeBuilderStore.setState({
      resume: {
        ...savedResume,
        volunteering: savedResume.volunteering.map((entry) => ({ ...entry, description })),
      },
    });

    const { container } = render(<ModernTemplate />);

    expect(screen.getByText("QA Volunteer")).toBeInTheDocument();
    expect(screen.queryByText("Describe your volunteering...")).not.toBeInTheDocument();
    expect(container.querySelector("p.whitespace-pre-line")).toBeNull();
  });

  it("allows the empty description to be edited in the desktop document", () => {
    useResumeBuilderStore.setState({
      resume: {
        ...savedResume,
        volunteering: savedResume.volunteering.map((entry) => ({ ...entry, description: "" })),
      },
      isDirty: false,
    });

    render(<ModernTemplate editable />);
    fireEvent.mouseDown(screen.getByRole("textbox", { name: "Describe your volunteering..." }));
    fireEvent.change(screen.getByRole("textbox", { name: "Describe your volunteering..." }), {
      target: { value: "Updated volunteering\nSecond line" },
    });
    fireEvent.blur(screen.getByRole("textbox", { name: "Describe your volunteering..." }));

    expect(useResumeBuilderStore.getState().resume?.volunteering[0].description).toBe(
      "Updated volunteering\nSecond line",
    );
    expect(screen.getByText(/Updated volunteering/).textContent).toBe("Updated volunteering\nSecond line");
  });

  it("preserves the description in the mobile text document", () => {
    useResumeBuilderStore.setState({
      resume: { ...savedResume, layout_mode: "double-left" },
    });

    render(
      <MobileTextEditingContext.Provider value>
        <ModernTemplate editable />
      </MobileTextEditingContext.Provider>,
    );

    expect(screen.getByRole("textbox", { name: "Describe your volunteering..." }).textContent).toBe(
      savedResume.volunteering[0].description,
    );
  });

  it("uses dark text while editing a sidebar description and preserves its value", () => {
    useResumeBuilderStore.setState({
      resume: {
        ...savedResume,
        layout_mode: "double-left",
        section_order: savedResume.section_order.map((section) => ({
          ...section,
          column: "sidebar",
        })),
      },
    });

    render(<ModernTemplate editable />);
    const field = screen.getByRole("textbox", { name: "Describe your volunteering..." });
    expect(field.tagName).toBe("P");
    expect(field).not.toHaveClass("text-gray-700");
    fireEvent.mouseDown(field);
    const textarea = screen.getByRole("textbox", { name: "Describe your volunteering..." });
    expect(textarea.tagName).toBe("TEXTAREA");
    expect(textarea).toHaveClass("[&:is(textarea)]:text-gray-700");
    fireEvent.change(textarea, { target: { value: "Updated sidebar description" } });
    fireEvent.blur(textarea);

    expect(useResumeBuilderStore.getState().resume?.volunteering[0].description).toBe(
      "Updated sidebar description",
    );
    expect(screen.getByText("Updated sidebar description").tagName).toBe("P");
  });

  it("renders markup in the description as text", () => {
    const description = '<img src="invalid" onerror="alert(1)">';
    useResumeBuilderStore.setState({
      resume: {
        ...savedResume,
        volunteering: savedResume.volunteering.map((entry) => ({ ...entry, description })),
      },
    });

    const { container } = render(<ModernTemplate />);

    expect(screen.getByText(description)).toBeInTheDocument();
    expect(container.querySelector("img")).toBeNull();
  });
});
