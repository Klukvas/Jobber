import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SummaryContent } from "./SummarySection";
import type { TemplateConfig } from "../templateConfig";
const config: TemplateConfig = {
  variant: "professional",
  summaryTitle: "Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: { renderAs: "pill", containerClass: "" },
  languages: { renderAs: "flex", containerClass: "" },
};

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number; max?: number }) =>
      `${key}:${options?.count ?? ""}:${options?.max ?? ""}`,
  }),
}));
function Summary({ initial = "" }: { initial?: string }) {
  const [content, setContent] = useState(initial);
  const setup = {
    summary: { content },
    updateSummary: (value: { content: string }) => setContent(value.content),
  } as unknown as Parameters<typeof SummaryContent>[0]["setup"];
  return <SummaryContent setup={setup} config={config} editable />;
}
describe("summary length validation", () => {
  it("limits new input to the 5000-character server limit with a visible count", async () => {
    render(<Summary initial={"x".repeat(4999)} />);
    await userEvent.click(screen.getByRole("textbox"));
    const editor = screen.getByRole("textbox");
    expect(editor).toHaveAttribute("maxlength", "5000");
    fireEvent.change(editor, { target: { value: "y".repeat(5000) } });
    fireEvent.change(editor, { target: { value: "y".repeat(5001) } });
    expect(editor).toHaveValue("y".repeat(5000));
    expect(
      screen.getByText("resumeBuilder.summaryLimit:5000:5000"),
    ).toBeInTheDocument();
  });
  it("explains existing over-limit data without truncating it", () => {
    render(<Summary initial={"x".repeat(5010)} />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "resumeBuilder.summaryTooLong",
    );
    expect(screen.getByRole("textbox")).toHaveTextContent("x".repeat(5010));
  });
});
