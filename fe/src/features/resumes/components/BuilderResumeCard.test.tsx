import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BuilderResumeCard } from "./BuilderResumeCard";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

function createBuilderResume(
  overrides?: Partial<ResumeBuilderDTO>,
): ResumeBuilderDTO {
  return {
    id: "b-1",
    title: "My Builder Resume",
    template_id: "professional",
    font_family: "Inter",
    primary_color: "#000",
    text_color: "#333",
    spacing: 150,
    margin_top: 40,
    margin_bottom: 40,
    margin_left: 40,
    margin_right: 40,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    skill_display: "",
    created_at: "2024-07-01T00:00:00Z",
    updated_at: "2024-07-10T00:00:00Z",
    ...overrides,
  };
}

const mockNavigate = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

vi.mock("@/features/resume-builder/components/ResumeThumbnail", () => ({
  ResumeThumbnail: () => <div data-testid="resume-thumbnail" />,
}));

describe("BuilderResumeCard", () => {
  const defaultProps = {
    resume: createBuilderResume(),
    limitReached: false,
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders resume title", () => {
    render(<BuilderResumeCard {...defaultProps} />);
    expect(screen.getByText("My Builder Resume")).toBeInTheDocument();
  });

  it("shows type badge 'Built'", () => {
    render(<BuilderResumeCard {...defaultProps} />);
    expect(screen.getByText("resumes.typeBuilt")).toBeInTheDocument();
  });

  it("renders resume thumbnail", () => {
    render(<BuilderResumeCard {...defaultProps} />);
    expect(screen.getByTestId("resume-thumbnail")).toBeInTheDocument();
  });

  it("shows last edited date", () => {
    render(<BuilderResumeCard {...defaultProps} />);
    expect(screen.getByText(/resumeBuilder.lastEdited/)).toBeInTheDocument();
  });

  it("navigates to editor when card is clicked", async () => {
    const user = userEvent.setup();
    render(<BuilderResumeCard {...defaultProps} />);

    await user.click(screen.getByText("My Builder Resume"));
    expect(mockNavigate).toHaveBeenCalledWith("/app/resume-builder/b-1");
  });

  it("calls onDuplicate when duplicate button clicked", async () => {
    const user = userEvent.setup();
    const onDuplicate = vi.fn();
    render(
      <BuilderResumeCard {...defaultProps} onDuplicate={onDuplicate} />,
    );

    await user.click(screen.getByLabelText("resumeBuilder.duplicate"));
    expect(onDuplicate).toHaveBeenCalledWith("b-1");
  });

  it("calls onDelete when delete button clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(<BuilderResumeCard {...defaultProps} onDelete={onDelete} />);

    await user.click(screen.getByLabelText("common.delete"));
    expect(onDelete).toHaveBeenCalledWith(defaultProps.resume);
  });

  it("disables duplicate button when limitReached is true", () => {
    render(<BuilderResumeCard {...defaultProps} limitReached={true} />);

    const btn = screen.getByLabelText("resumeBuilder.duplicate");
    expect(btn).toBeDisabled();
  });

  it("shows limit tooltip when duplicate is disabled", () => {
    render(<BuilderResumeCard {...defaultProps} limitReached={true} />);

    const btn = screen.getByLabelText("resumeBuilder.duplicate");
    expect(btn).toHaveAttribute(
      "title",
      "settings.subscription.limitReached",
    );
  });

  it("does not show limit tooltip when duplicate is enabled", () => {
    render(<BuilderResumeCard {...defaultProps} limitReached={false} />);

    const btn = screen.getByLabelText("resumeBuilder.duplicate");
    expect(btn).not.toHaveAttribute("title");
  });
});
