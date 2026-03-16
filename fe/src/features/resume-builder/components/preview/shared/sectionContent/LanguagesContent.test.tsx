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
  EditableSelect: ({ value }: { value: string }) => (
    <span data-testid="select">{value}</span>
  ),
}));
vi.mock("../../../inline/EntryWrapper", () => ({
  EntryWrapper: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="entry-wrapper">{children}</div>
  ),
}));

import { LanguagesContent } from "./LanguagesContent";
import type { TemplateConfig } from "../templateConfig";

function makeConfig(variant: TemplateConfig["variant"]): TemplateConfig {
  return {
    variant,
    summaryTitle: "Summary",
    textSize: "text-xs",
    leadingClass: "leading-relaxed",
    skills: { renderAs: "pill", containerClass: "" },
    languages: {
      renderAs: "flex",
      containerClass: "flex flex-wrap gap-x-4 gap-y-1",
    },
  };
}

function makeSetup(languages: Array<Record<string, unknown>> = []) {
  return {
    resume: {
      languages,
      primary_color: "#e11d48",
      text_color: "#e11d48",
      section_order: [],
    },
    color: "#e11d48",
    textColor: "#e11d48",
    updateLanguage: vi.fn(),
    languagesSection: { handleAdd: vi.fn(), handleRemove: vi.fn() },
  } as unknown as Parameters<typeof LanguagesContent>[0]["setup"];
}

const sampleLang = { id: "l1", name: "English", proficiency: "native" };
const sampleLang2 = {
  id: "l2",
  name: "Spanish",
  proficiency: "professional_working",
};

describe("LanguagesContent", () => {
  // ---------------------------------------------------------------------------
  // Minimal variant
  // ---------------------------------------------------------------------------

  describe("minimal variant", () => {
    it("renders individual entries when editable", () => {
      const setup = makeSetup([sampleLang, sampleLang2]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={true}
        />,
      );
      expect(screen.getAllByTestId("entry-wrapper").length).toBe(2);
      expect(screen.getAllByTestId("field-Language").length).toBe(2);
    });

    it("shows proficiency in parentheses when editable and proficiency exists", () => {
      const setup = makeSetup([sampleLang]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={true}
        />,
      );
      expect(screen.getByText(/\(native\)/)).toBeInTheDocument();
    });

    it("joins languages with dot separator in view mode", () => {
      const setup = makeSetup([sampleLang, sampleLang2]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(
        screen.getByText("English (native) · Spanish (professional_working)"),
      ).toBeInTheDocument();
    });

    it("returns null when no languages in view mode", () => {
      const setup = makeSetup([]);
      const { container } = render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(container.innerHTML).toBe("");
    });

    it("omits proficiency from joined text when empty", () => {
      const setup = makeSetup([{ id: "l1", name: "English", proficiency: "" }]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("minimal")}
          editable={false}
        />,
      );
      expect(screen.getByText("English")).toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Modern variant
  // ---------------------------------------------------------------------------

  describe("modern variant", () => {
    it("renders proficiency with opacity-70 class", () => {
      const setup = makeSetup([sampleLang]);
      const { container } = render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      const opacitySpan = container.querySelector(".opacity-70");
      expect(opacitySpan).toBeTruthy();
    });

    it("uses em-dash separator before proficiency", () => {
      const setup = makeSetup([sampleLang]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(screen.getByText(/\u2014/)).toBeInTheDocument();
    });

    it("hides proficiency when empty and not editable", () => {
      const setup = makeSetup([{ id: "l1", name: "English", proficiency: "" }]);
      const { container } = render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("modern")}
          editable={false}
        />,
      );
      expect(container.querySelector(".opacity-70")).not.toBeInTheDocument();
    });
  });

  // ---------------------------------------------------------------------------
  // Default variant
  // ---------------------------------------------------------------------------

  describe("default variant", () => {
    it("renders language name and proficiency select", () => {
      const setup = makeSetup([sampleLang]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByTestId("field-Language")).toHaveTextContent("English");
      expect(screen.getByTestId("select")).toHaveTextContent("native");
    });

    it("uses em-dash separator", () => {
      const setup = makeSetup([sampleLang]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.getByText(/\u2014/)).toBeInTheDocument();
    });

    it("hides proficiency when empty and not editable", () => {
      const setup = makeSetup([{ id: "l1", name: "English", proficiency: "" }]);
      render(
        <LanguagesContent
          setup={setup}
          config={makeConfig("professional")}
          editable={false}
        />,
      );
      expect(screen.queryByTestId("select")).not.toBeInTheDocument();
    });
  });
});
