import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

/**
 * SummaryContent now only renders the body (textarea / text paragraph).
 * The heading is handled by SectionHeader via TemplateSections.
 */

// Mock EditableTextarea to simplify rendering
vi.mock("../../../inline/EditableTextarea", () => ({
  EditableTextarea: ({ value }: { value: string }) => <span>{value}</span>,
}));

import { SummaryContent } from "./SummarySection";
import type { TemplateConfig } from "../templateConfig";

function makeConfig(variant: string): TemplateConfig {
  return {
    variant: variant as TemplateConfig["variant"],
    summaryTitle: "About Me",
    textSize: "text-xs",
    leadingClass: "leading-relaxed",
    skills: { renderAs: "pill", containerClass: "flex flex-wrap gap-1.5" },
    languages: {
      renderAs: "flex",
      containerClass: "flex flex-wrap gap-x-4 gap-y-1",
    },
    summaryMb: "mb-4",
  };
}

const baseSetup = {
  summary: { content: "Test summary content" },
  color: "#e11d48",
  textColor: "#e11d48",
  hideSection: vi.fn(),
  updateSummary: vi.fn(),
} as unknown as Parameters<typeof SummaryContent>[0]["setup"];

describe("SummaryContent", () => {
  it("renders summary content text when not editable", () => {
    const config = makeConfig("bold");
    render(
      <SummaryContent setup={baseSetup} config={config} editable={false} />,
    );
    expect(screen.getByText("Test summary content")).toBeInTheDocument();
  });

  it("renders EditableTextarea when editable", () => {
    const config = makeConfig("professional");
    render(
      <SummaryContent setup={baseSetup} config={config} editable={true} />,
    );
    // EditableTextarea is mocked as <span>{value}</span>
    expect(screen.getByText("Test summary content")).toBeInTheDocument();
  });

  it("renders nothing when not editable and no content", () => {
    const emptySetup = {
      ...baseSetup,
      summary: { content: "" },
    } as unknown as Parameters<typeof SummaryContent>[0]["setup"];

    const config = makeConfig("professional");
    const { container } = render(
      <SummaryContent setup={emptySetup} config={config} editable={false} />,
    );
    expect(container.innerHTML).toBe("");
  });
});
