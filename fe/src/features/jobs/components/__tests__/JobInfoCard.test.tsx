import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { JobInfoCard } from "../JobInfoCard";
import type { JobDTO } from "@/shared/types/api";
import type { EditableFields } from "@/features/jobs/hooks/useJobDetailMutations";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

// Isolate from CompanySelectWithQuickAdd's react-query dependency.
vi.mock("@/features/jobs/components/CompanySelectWithQuickAdd", () => ({
  CompanySelectWithQuickAdd: ({ value }: { value: string }) => (
    <div data-testid="company-select">company:{value}</div>
  ),
}));

function makeJob(overrides: Partial<JobDTO> = {}): JobDTO {
  return {
    id: "job-1",
    title: "Engineer",
    is_archived: false,
    current_stage_name: "Screening",
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T00:00:00Z",
    ...overrides,
  } as JobDTO;
}

function makeFields(overrides: Partial<EditableFields> = {}): EditableFields {
  return {
    title: "Engineer",
    company_id: "c1",
    url: "",
    source: "",
    description: "",
    notes: "",
    ...overrides,
  };
}

describe("JobInfoCard", () => {
  it("shows the current stage name badge when not archived", () => {
    render(
      <JobInfoCard
        job={makeJob({ current_stage_name: "Interview" })}
        fields={makeFields()}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    expect(screen.getByText("Interview")).toBeInTheDocument();
  });

  it("shows the archived badge when the job is archived", () => {
    render(
      <JobInfoCard
        job={makeJob({ is_archived: true })}
        fields={makeFields()}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    expect(screen.getByText("jobs.archived")).toBeInTheDocument();
  });

  it("falls back to a no-stage label when there is no current stage", () => {
    render(
      <JobInfoCard
        job={makeJob({ current_stage_name: undefined })}
        fields={makeFields()}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    expect(screen.getByText("jobs.board.noStage")).toBeInTheDocument();
  });

  it("emits field changes as the user edits the title and source", async () => {
    const user = userEvent.setup();
    const onFieldChange = vi.fn();
    render(
      <JobInfoCard
        job={makeJob()}
        fields={makeFields({ title: "" })}
        companies={[]}
        onFieldChange={onFieldChange}
      />,
    );

    await user.type(screen.getByPlaceholderText("jobs.titlePlaceholder"), "A");
    expect(onFieldChange).toHaveBeenCalledWith("title", "A");

    await user.type(screen.getByLabelText("jobs.source"), "X");
    expect(onFieldChange).toHaveBeenCalledWith("source", "X");
  });

  it("renders an external link only for a valid http(s) url", () => {
    const { rerender } = render(
      <JobInfoCard
        job={makeJob()}
        fields={makeFields({ url: "https://jobs.example.com/1" })}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://jobs.example.com/1");

    rerender(
      <JobInfoCard
        job={makeJob()}
        fields={makeFields({ url: "not-a-url" })}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("does not render a link for a non-http protocol", () => {
    render(
      <JobInfoCard
        job={makeJob()}
        fields={makeFields({ url: "javascript:alert(1)" })}
        companies={[]}
        onFieldChange={vi.fn()}
      />,
    );
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
