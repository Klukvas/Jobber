import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  JobMobileAccordion,
  type MobileColumnData,
} from "../JobMobileAccordion";
import type { JobDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

// Keep the card trivial — we only care about accordion open/close behavior.
vi.mock("../JobCardBase", () => ({
  JobCardBase: ({ job }: { job: JobDTO }) => <div>{job.title}</div>,
}));

function makeJob(id: string, title: string): JobDTO {
  return {
    id,
    title,
    company_name: "Acme",
    is_archived: false,
    is_favorite: false,
    current_stage_name: "Applied",
    last_activity_at: "2026-01-10T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-10T00:00:00Z",
  };
}

function makeColumns(): MobileColumnData[] {
  return [
    { id: "applied", label: "Applied", jobs: [makeJob("j1", "Job One")] },
    { id: "interview", label: "Interview", jobs: [makeJob("j2", "Job Two")] },
  ];
}

function renderAccordion() {
  return render(
    <JobMobileAccordion
      columns={makeColumns()}
      onAddComment={vi.fn()}
      onAddStage={vi.fn()}
      onDelete={vi.fn()}
    />,
  );
}

const header = (name: RegExp) => screen.getByRole("button", { name });

describe("JobMobileAccordion", () => {
  it("opens the first non-empty column by default", () => {
    renderAccordion();
    expect(header(/Applied/)).toHaveAttribute("aria-expanded", "true");
    expect(header(/Interview/)).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Job One")).toBeInTheDocument();
    expect(screen.queryByText("Job Two")).not.toBeInTheDocument();
  });

  it("collapses an open stage and keeps it collapsed", async () => {
    const user = userEvent.setup();
    renderAccordion();

    await user.click(header(/Applied/));

    // Regression: closing the last open stage must NOT snap it back open.
    expect(header(/Applied/)).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Job One")).not.toBeInTheDocument();
  });

  it("expands a collapsed stage when its header is tapped", async () => {
    const user = userEvent.setup();
    renderAccordion();

    await user.click(header(/Interview/));

    expect(header(/Interview/)).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Job Two")).toBeInTheDocument();
  });
});
