import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("../../../inline/EditableField", () => ({
  EditableField: ({
    value,
    placeholder,
    style,
    className,
  }: {
    value: string;
    placeholder: string;
    style?: React.CSSProperties;
    className?: string;
  }) => (
    <span
      data-testid={`field-${placeholder}`}
      style={style}
      className={className}
    >
      {value}
    </span>
  ),
}));
vi.mock("../../../inline/EditableDateRange", () => ({
  EditableDateRange: ({ className }: { className?: string }) => (
    <span data-testid="date-range" className={className}>
      2018 – 2022
    </span>
  ),
}));
vi.mock("../../../inline/EntryWrapper", () => ({
  EntryWrapper: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => (
    <div data-testid="entry-wrapper" className={className}>
      {children}
    </div>
  ),
}));

import { EducationContent } from "./EducationContent";
import type { TemplateConfig } from "../templateConfig";

function makeConfig(
  variant: TemplateConfig["variant"],
  overrides?: Partial<TemplateConfig>,
): TemplateConfig {
  return {
    variant,
    summaryTitle: "Summary",
    textSize: "text-xs",
    leadingClass: "leading-relaxed",
    skills: { renderAs: "pill", containerClass: "" },
    languages: { renderAs: "flex", containerClass: "" },
    ...overrides,
  };
}

function makeSetup(
  educations: Array<Record<string, unknown>> = [],
  overrides?: Record<string, unknown>,
) {
  return {
    resume: {
      educations,
      primary_color: "#e11d48",
      text_color: "#e11d48",
      section_order: [],
    },
    color: "#e11d48",
    textColor: "#e11d48",
    updateEducation: vi.fn(),
    educationSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    ...overrides,
  } as unknown as Parameters<typeof EducationContent>[0]["setup"];
}

const sampleEdu = {
  id: "edu-1",
  degree: "Bachelor of Science",
  field_of_study: "Computer Science",
  institution: "MIT",
  start_date: "2018-09",
  end_date: "2022-05",
  is_current: false,
  gpa: "3.9",
};

describe("EducationContent", () => {
  // ---------------------------------------------------------------------------
  // Minimal variant
  // ---------------------------------------------------------------------------

  describe("minimal variant", () => {
    it("renders degree and field with comma separator", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Degree")).toHaveTextContent(
        "Bachelor of Science",
      );
      expect(screen.getByTestId("field-Field of Study")).toHaveTextContent(
        "Computer Science",
      );
      const heading = screen.getByTestId("field-Degree").closest("p");
      expect(heading!.textContent).toContain(", ");
    });

    it("hides field_of_study when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleEdu, field_of_study: "" }]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(
        screen.queryByTestId("field-Field of Study"),
      ).not.toBeInTheDocument();
    });

    it("does not render GPA in minimal variant", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.queryByText("GPA:")).not.toBeInTheDocument();
    });

    it("renders institution with dot separator", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Institution")).toHaveTextContent("MIT");
      expect(screen.getByText(/·/)).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Non-minimal variants
  // ---------------------------------------------------------------------------

  describe("non-minimal (professional) variant", () => {
    it("renders degree and field with 'in' separator", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Degree")).toHaveTextContent(
        "Bachelor of Science",
      );
      expect(screen.getByTestId("field-Field of Study")).toHaveTextContent(
        "Computer Science",
      );
      expect(screen.getByText(/in/)).toBeInTheDocument();
    });

    it("renders GPA when present", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByText("GPA:")).toBeInTheDocument();
      expect(screen.getByTestId("field-3.8")).toHaveTextContent("3.9");
    });

    it("hides GPA when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleEdu, gpa: "" }]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.queryByText("GPA:")).not.toBeInTheDocument();
    });

    it("shows GPA when empty but editable", () => {
      const setup = makeSetup([{ ...sampleEdu, gpa: "" }]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={true}
        />,
      );
      expect(screen.getByText("GPA:")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Modern variant - institution color
  // ---------------------------------------------------------------------------

  describe("modern variant", () => {
    it("applies effectiveColor to institution", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      const institution = screen.getByTestId("field-Institution");
      expect(institution.style.color).toBe("rgb(225, 29, 72)");
    });

    it("uses sectionColor over setup.color when provided", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
          sectionColor="#3b82f6"
        />,
      );
      const institution = screen.getByTestId("field-Institution");
      expect(institution.style.color).toBe("rgb(59, 130, 246)");
    });

    it("non-modern institution has gray-600 class", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      const institution = screen.getByTestId("field-Institution");
      expect(institution.className).toContain("text-gray-600");
    });
  });

  // ---------------------------------------------------------------------------
  // Entry spacing
  // ---------------------------------------------------------------------------

  describe("entry spacing", () => {
    it("uses default mb-2 spacing", () => {
      const setup = makeSetup([sampleEdu]);
      render(
        <EducationContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("entry-wrapper").className).toContain("mb-2");
    });
  });
});
