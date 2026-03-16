import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

// Mock inline components
vi.mock("../../../inline/EditableField", () => ({
  EditableField: ({
    value,
    placeholder,
    as: Tag = "span",
    className,
  }: {
    value: string;
    placeholder: string;
    as?: string;
    className?: string;
  }) => {
    const El = Tag as keyof JSX.IntrinsicElements;
    return (
      <El data-testid={`field-${placeholder}`} className={className}>
        {value}
      </El>
    );
  },
}));
vi.mock("../../../inline/EditableTextarea", () => ({
  EditableTextarea: ({
    value,
    className,
  }: {
    value: string;
    className?: string;
  }) => (
    <span data-testid="textarea" className={className}>
      {value}
    </span>
  ),
}));
vi.mock("../../../inline/EditableDateRange", () => ({
  EditableDateRange: ({ className }: { className?: string }) => (
    <span data-testid="date-range" className={className}>
      Jan 2020 – Present
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

import { ExperienceContent } from "./ExperienceContent";
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
    skills: { renderAs: "pill", containerClass: "flex flex-wrap gap-1.5" },
    languages: {
      renderAs: "flex",
      containerClass: "flex flex-wrap gap-x-4 gap-y-1",
    },
    ...overrides,
  };
}

function makeSetup(
  experiences: Array<Record<string, unknown>> = [],
  overrides?: Record<string, unknown>,
) {
  return {
    resume: {
      experiences,
      primary_color: "#e11d48",
      text_color: "#e11d48",
      section_order: [],
    },
    color: "#e11d48",
    textColor: "#e11d48",
    updateExperience: vi.fn(),
    experienceSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
    ...overrides,
  } as unknown as Parameters<typeof ExperienceContent>[0]["setup"];
}

const sampleExp = {
  id: "exp-1",
  position: "Software Engineer",
  company: "Acme Inc",
  location: "NYC",
  start_date: "2020-01",
  end_date: "",
  is_current: true,
  description: "Built amazing things",
};

// ---------------------------------------------------------------------------
// Minimal variant
// ---------------------------------------------------------------------------

describe("ExperienceContent", () => {
  describe("minimal variant", () => {
    it("renders position and company with 'at' separator", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Position")).toHaveTextContent(
        "Software Engineer",
      );
      expect(screen.getByTestId("field-Company")).toHaveTextContent("Acme Inc");
      expect(screen.getByText(/at/)).toBeInTheDocument();
    });

    it("hides company when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleExp, company: "" }]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("field-Company")).not.toBeInTheDocument();
    });

    it("shows company when empty but editable", () => {
      const setup = makeSetup([{ ...sampleExp, company: "" }]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={true}
        />,
      );
      expect(screen.getByTestId("field-Company")).toBeInTheDocument();
    });

    it("renders description when present", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("textarea")).toHaveTextContent(
        "Built amazing things",
      );
    });

    it("hides description when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleExp, description: "" }]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("textarea")).not.toBeInTheDocument();
    });

    it("applies date range with correct classes", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      const dateRange = screen.getByTestId("date-range");
      expect(dateRange.className).toContain("text-xs");
      expect(dateRange.className).toContain("text-gray-400");
    });
  });

  // ---------------------------------------------------------------------------
  // Modern variant
  // ---------------------------------------------------------------------------

  describe("modern variant", () => {
    it("renders company with effectiveColor", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      const companyContainer = screen.getByTestId("field-Company").closest("p");
      expect(companyContainer).toBeTruthy();
      expect(companyContainer!.style.color).toBe("rgb(225, 29, 72)");
    });

    it("uses sectionColor when provided", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
          sectionColor="#3b82f6"
        />,
      );
      const companyContainer = screen.getByTestId("field-Company").closest("p");
      expect(companyContainer!.style.color).toBe("rgb(59, 130, 246)");
    });

    it("shows location with comma separator when both exist", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Location")).toHaveTextContent("NYC");
      const companyContainer = screen.getByTestId("field-Company").closest("p");
      expect(companyContainer!.textContent).toContain(", ");
    });

    it("hides location when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleExp, location: "" }]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("field-Location")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Default variant (professional, etc.)
  // ---------------------------------------------------------------------------

  describe("default variant", () => {
    it("renders company and location with em-dash separator", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Company")).toHaveTextContent("Acme Inc");
      expect(screen.getByTestId("field-Location")).toHaveTextContent("NYC");
      expect(screen.getByText(/\u2014/)).toBeInTheDocument();
    });

    it("renders position as bold", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      const positionField = screen.getByTestId("field-Position");
      expect(positionField.className).toContain("font-bold");
    });

    it("renders description with gray-700 text", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      const textarea = screen.getByTestId("textarea");
      expect(textarea.className).toContain("text-gray-700");
    });

    it("hides location when empty and not editable", () => {
      const setup = makeSetup([{ ...sampleExp, location: "" }]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("field-Location")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Entry spacing
  // ---------------------------------------------------------------------------

  describe("entry spacing", () => {
    it("uses default mb-3 when no entrySpacing override", () => {
      const setup = makeSetup([sampleExp]);
      render(
        <ExperienceContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("entry-wrapper").className).toContain("mb-3");
    });

    it("uses custom entrySpacing when provided", () => {
      const setup = makeSetup([sampleExp]);
      const config = makeConfig("professional", {
        entrySpacing: { experience: "mb-6" },
      });
      render(
        <ExperienceContent setup={setup} config={config} editable={false} />,
      );
      expect(screen.getByTestId("entry-wrapper").className).toContain("mb-6");
    });
  });
});
