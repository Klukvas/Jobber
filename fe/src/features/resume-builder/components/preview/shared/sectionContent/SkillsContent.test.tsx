import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("../../../inline/EditableField", () => ({
  EditableField: ({
    value,
    placeholder,
  }: {
    value: string;
    placeholder: string;
  }) => <span data-testid={`field-${placeholder}`}>{value}</span>,
}));
vi.mock("../../../inline/EditableSelect", () => ({
  EditableSelect: ({
    value,
    editable,
  }: {
    value: string;
    editable: boolean;
  }) =>
    editable ? (
      <span data-testid="select">{value}</span>
    ) : (
      <span data-testid="select">{value}</span>
    ),
}));
vi.mock("../../../inline/EntryWrapper", () => ({
  EntryWrapper: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="entry-wrapper">{children}</div>
  ),
}));

import { SkillsContent } from "./SkillsContent";
import type { TemplateConfig } from "../templateConfig";
import { MAX_DOTS, MAX_SKILL_LEVEL } from "./skillLevelUtils";

function makeConfig(
  renderAs: string,
  overrides?: Partial<TemplateConfig>,
): TemplateConfig {
  return {
    variant: "professional",
    summaryTitle: "Summary",
    textSize: "text-xs",
    leadingClass: "leading-relaxed",
    skills: {
      renderAs: renderAs as TemplateConfig["skills"]["renderAs"],
      containerClass: "flex flex-wrap gap-1.5",
    },
    languages: { renderAs: "flex", containerClass: "" },
    ...overrides,
  };
}

function makeSetup(skills: Array<Record<string, unknown>> = []) {
  return {
    resume: {
      skills,
      primary_color: "#e11d48",
      text_color: "#e11d48",
      section_order: [],
    },
    color: "#e11d48",
    textColor: "#e11d48",
    updateSkill: vi.fn(),
    skillsSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
  } as unknown as Parameters<typeof SkillsContent>[0]["setup"];
}

const basicSkill = { id: "s1", name: "React", level: "advanced" };
const expertSkill = { id: "s2", name: "TypeScript", level: "expert" };

describe("SkillsContent", () => {
  // ---------------------------------------------------------------------------
  // pill
  // ---------------------------------------------------------------------------

  describe("pill mode", () => {
    it("renders pill with colored background and white text class", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("pill")}
          editable={false}
        />,
      );
      const pill = container.querySelector(".rounded-full");
      expect(pill).toBeTruthy();
      expect(pill!.className).toContain("text-white");
      expect((pill as HTMLElement).style.backgroundColor).toBe(
        "rgb(225, 29, 72)",
      );
    });

    it("uses sectionColor for background when provided", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("pill")}
          editable={false}
          sectionColor="#3b82f6"
        />,
      );
      const pill = container.querySelector(".rounded-full");
      expect((pill as HTMLElement).style.backgroundColor).toBe(
        "rgb(59, 130, 246)",
      );
    });
  });

  // ---------------------------------------------------------------------------
  // dots
  // ---------------------------------------------------------------------------

  describe("dots mode", () => {
    it("renders MAX_DOTS dots total", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("dots")}
          editable={false}
        />,
      );
      const dots = container.querySelectorAll(".rounded-full.inline-block");
      expect(dots.length).toBe(MAX_DOTS);
    });

    it("fills correct number based on skill level (advanced = 3)", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("dots")}
          editable={false}
        />,
      );
      const dots = container.querySelectorAll(".rounded-full.inline-block");
      const filled = Array.from(dots).filter(
        (d) => (d as HTMLElement).style.backgroundColor !== "",
      );
      expect(filled.length).toBe(3); // advanced = 3
    });

    it("shows level select only in editable mode", () => {
      const setup = makeSetup([basicSkill]);
      const { rerender } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("dots")}
          editable={false}
        />,
      );
      expect(screen.queryAllByTestId("select").length).toBe(0);

      rerender(
        <SkillsContent
          setup={setup}
          config={makeConfig("dots")}
          editable={true}
        />,
      );
      expect(screen.getAllByTestId("select").length).toBe(1);
    });
  });

  // ---------------------------------------------------------------------------
  // bar
  // ---------------------------------------------------------------------------

  describe("bar mode", () => {
    it("renders bar with correct width percentage", () => {
      const setup = makeSetup([basicSkill]); // advanced = 3
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("bar")}
          editable={false}
        />,
      );
      const bar = container.querySelector(".rounded-full.transition-all");
      const expectedWidth = (3 / MAX_SKILL_LEVEL) * 100;
      expect((bar as HTMLElement).style.width).toBe(`${expectedWidth}%`);
    });

    it("renders bar with effectiveColor background", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("bar")}
          editable={false}
        />,
      );
      const bar = container.querySelector(".rounded-full.transition-all");
      expect((bar as HTMLElement).style.backgroundColor).toBe(
        "rgb(225, 29, 72)",
      );
    });
  });

  // ---------------------------------------------------------------------------
  // text-only
  // ---------------------------------------------------------------------------

  describe("text-only mode", () => {
    it("renders joined text with dots in view mode", () => {
      const setup = makeSetup([basicSkill, expertSkill]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("text-only")}
          editable={false}
        />,
      );
      expect(screen.getByText("React · TypeScript")).toBeInTheDocument();
    });

    it("returns null when no skills in view mode", () => {
      const setup = makeSetup([]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("text-only")}
          editable={false}
        />,
      );
      expect(container.innerHTML).toBe("");
    });

    it("renders individual entries in editable mode", () => {
      const setup = makeSetup([basicSkill, expertSkill]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("text-only")}
          editable={true}
        />,
      );
      expect(screen.getAllByTestId("field-Skill").length).toBe(2);
      expect(screen.getAllByTestId("entry-wrapper").length).toBe(2);
    });
  });

  // ---------------------------------------------------------------------------
  // circle (SVG)
  // ---------------------------------------------------------------------------

  describe("circle mode", () => {
    it("renders SVG with correct dashoffset for advanced level", () => {
      const setup = makeSetup([basicSkill]); // advanced = 3
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("circle")}
          editable={false}
        />,
      );
      const circles = container.querySelectorAll("circle");
      expect(circles.length).toBe(2); // background + foreground
      const foreground = circles[1];
      const radius = 14;
      const circumference = 2 * Math.PI * radius;
      const pct = (3 / MAX_SKILL_LEVEL) * 100;
      const expectedOffset = circumference - (circumference * pct) / 100;
      expect(Number(foreground.getAttribute("stroke-dashoffset"))).toBeCloseTo(
        expectedOffset,
      );
    });

    it("shows percentage text in SVG", () => {
      const setup = makeSetup([basicSkill]); // advanced = 3
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("circle")}
          editable={false}
        />,
      );
      const pct = (3 / MAX_SKILL_LEVEL) * 100;
      expect(screen.getByText(`${pct}%`)).toBeInTheDocument();
    });

    it("shows dash when level is 0", () => {
      const setup = makeSetup([{ id: "s1", name: "React", level: "" }]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("circle")}
          editable={false}
        />,
      );
      expect(screen.getByText("–")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // star
  // ---------------------------------------------------------------------------

  describe("star mode", () => {
    it("renders MAX_DOTS stars with correct coloring", () => {
      const setup = makeSetup([basicSkill]); // advanced = 3
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("star")}
          editable={false}
        />,
      );
      const stars = container.querySelectorAll(".leading-none");
      expect(stars.length).toBe(MAX_DOTS);
      const colored = Array.from(stars).filter(
        (s) => (s as HTMLElement).style.color === "rgb(225, 29, 72)",
      );
      expect(colored.length).toBe(3);
    });
  });

  // ---------------------------------------------------------------------------
  // square
  // ---------------------------------------------------------------------------

  describe("square mode", () => {
    it("renders MAX_DOTS squares with rounded-sm", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("square")}
          editable={false}
        />,
      );
      const squares = container.querySelectorAll(".rounded-sm.inline-block");
      expect(squares.length).toBe(MAX_DOTS);
    });
  });

  // ---------------------------------------------------------------------------
  // segmented
  // ---------------------------------------------------------------------------

  describe("segmented mode", () => {
    it("renders MAX_SKILL_LEVEL segments", () => {
      const setup = makeSetup([basicSkill]);
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("segmented")}
          editable={false}
        />,
      );
      const segments = container.querySelectorAll(".rounded-sm");
      expect(segments.length).toBe(MAX_SKILL_LEVEL);
    });
  });

  // ---------------------------------------------------------------------------
  // bubble
  // ---------------------------------------------------------------------------

  describe("bubble mode", () => {
    it("renders with color-mix background based on level", () => {
      const setup = makeSetup([basicSkill]); // advanced = 3
      const { container } = render(
        <SkillsContent
          setup={setup}
          config={makeConfig("bubble")}
          editable={false}
        />,
      );
      const bubble = container.querySelector(".rounded-lg");
      expect(bubble).toBeTruthy();
      const style = (bubble as HTMLElement).style;
      expect(style.color).toBe("rgb(225, 29, 72)");
      expect(style.backgroundColor).toContain("color-mix");
    });

    it("shows level separator in bubble when level exists", () => {
      const setup = makeSetup([basicSkill]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("bubble")}
          editable={false}
        />,
      );
      expect(screen.getByText("|")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // text-level
  // ---------------------------------------------------------------------------

  describe("text-level mode", () => {
    it("renders skill name with level in parentheses", () => {
      const setup = makeSetup([basicSkill]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("text-level")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Skill name")).toHaveTextContent("React");
      const entry = screen.getByTestId("entry-wrapper");
      expect(entry.textContent).toContain("(");
      expect(entry.textContent).toContain(")");
    });

    it("hides level when empty and not editable", () => {
      const setup = makeSetup([{ id: "s1", name: "React", level: "" }]);
      render(
        <SkillsContent
          setup={setup}
          config={makeConfig("text-level")}
          editable={false}
        />,
      );
      expect(screen.queryByText("(")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // default case
  // ---------------------------------------------------------------------------

  describe("unknown render mode", () => {
    it("returns null for unknown render mode", () => {
      const setup = makeSetup([basicSkill]);
      const config = makeConfig("unknown-mode" as string);
      const { container } = render(
        <SkillsContent setup={setup} config={config} editable={false} />,
      );
      expect(container.innerHTML).toBe("");
    });
  });
});
