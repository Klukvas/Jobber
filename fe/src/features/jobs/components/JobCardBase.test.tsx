import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { JobCardBase } from "./JobCardBase";
import type { JobDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const job: JobDTO = {
  id: "job-1",
  title: "Backend Engineer",
  is_archived: false,
  is_favorite: false,
  company_name: "Acme",
  source: "LinkedIn",
  current_stage_template_id: "tpl-1",
  current_stage_name: "Screening",
  last_activity_at: "2026-08-01T00:00:00Z",
  created_at: "2026-07-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
};

function renderCard(
  overrides: Partial<Parameters<typeof JobCardBase>[0]> = {},
) {
  const props = {
    job,
    onTitleClick: vi.fn(),
    onAddComment: vi.fn(),
    onAddStage: vi.fn(),
    onDelete: vi.fn(),
    ...overrides,
  };
  render(<JobCardBase {...props} />);
  return props;
}

describe("JobCardBase — content", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows the title, company, and source (stage is conveyed by its column)", () => {
    renderCard();
    expect(screen.getByText("Backend Engineer")).toBeInTheDocument();
    expect(screen.getByText("Acme")).toBeInTheDocument();
    expect(screen.getByText("LinkedIn")).toBeInTheDocument();
    // The stage name lives in the column header, not on the card.
    expect(screen.queryByText("Screening")).not.toBeInTheDocument();
  });
});

describe("JobCardBase — actions menu", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not show the menu until the actions button is clicked", () => {
    renderCard();
    expect(screen.queryByText("jobs.delete")).not.toBeInTheDocument();
  });

  it("shows Add Comment, Add Stage, and Delete in the menu", () => {
    renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    expect(screen.getByText("jobs.addComment")).toBeInTheDocument();
    expect(screen.getByText("jobs.addStage")).toBeInTheDocument();
    expect(screen.getByText("jobs.delete")).toBeInTheDocument();
  });

  it("calls onDelete with the job and closes the menu when Delete is clicked", () => {
    const { onDelete } = renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.delete"));
    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(onDelete).toHaveBeenCalledWith(job);
    // menu collapses after selecting an item
    expect(screen.queryByText("jobs.delete")).not.toBeInTheDocument();
  });

  it("calls onAddComment and onAddStage from the menu", () => {
    const { onAddComment, onAddStage, onDelete } = renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.addComment"));
    expect(onAddComment).toHaveBeenCalledWith(job);

    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.addStage"));
    expect(onAddStage).toHaveBeenCalledWith(job);

    expect(onDelete).not.toHaveBeenCalled();
  });
});

/**
 * Measured on a 390px viewport: the title button was 242x18 and the actions
 * button 32x32, both well under the 44x44 WCAG 2.5.5 and the Apple HIG ask
 * for. Asserted on classes because jsdom has no layout, and the classes are
 * what decide the size; the floor is scoped to `max-sm` so the board keeps its
 * pointer density.
 */
describe("JobCardBase — touch targets", () => {
  beforeEach(() => vi.clearAllMocks());

  it("gives the title button a 44px height on phones", () => {
    renderCard();
    const title = screen.getByText("Backend Engineer").closest("button");

    expect(title?.className).toMatch(/max-sm:min-h-11/);
  });

  it("gives the actions button a 44x44 target on phones", () => {
    renderCard();
    const actions = screen.getByLabelText("jobs.actionsMenu");

    expect(actions.className).toMatch(/max-sm:h-11/);
    expect(actions.className).toMatch(/max-sm:w-11/);
  });

  it("keeps the actions button at its drawn 32x32 on a pointer", () => {
    renderCard();
    const actions = screen.getByLabelText("jobs.actionsMenu");

    expect(actions.className).toMatch(/(^|\s)h-8(\s|$)/);
    expect(actions.className).toMatch(/(^|\s)w-8(\s|$)/);
  });
});
