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
  status: "applied",
  is_favorite: false,
  company_name: "Acme",
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
    onChangeStatus: vi.fn(),
    onDelete: vi.fn(),
    ...overrides,
  };
  render(<JobCardBase {...props} />);
  return props;
}

describe("JobCardBase — delete action", () => {
  beforeEach(() => vi.clearAllMocks());

  it("does not show the menu (incl. delete) until the actions button is clicked", () => {
    renderCard();
    expect(screen.queryByText("jobs.delete")).not.toBeInTheDocument();
  });

  it("shows a Delete item in the actions menu", () => {
    renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
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

  it("does not trigger delete when other menu items are used", () => {
    const { onDelete, onAddComment } = renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.addComment"));
    expect(onAddComment).toHaveBeenCalledWith(job);
    expect(onDelete).not.toHaveBeenCalled();
  });
});

describe("JobCardBase — quick status submenu", () => {
  beforeEach(() => vi.clearAllMocks());

  it("expands an inline status list and picks a status with one click", () => {
    const onStatusSelect = vi.fn();
    const { onChangeStatus } = renderCard({ onStatusSelect });

    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.changeStatus"));

    // current status (applied) is not offered in the submenu (the card badge
    // outside the menu still shows it)
    const menuItems = screen.getAllByRole("menuitem");
    expect(
      menuItems.some((item) =>
        item.textContent?.includes("jobs.statusApplied"),
      ),
    ).toBe(false);
    fireEvent.click(screen.getByText("jobs.statusRejected"));

    expect(onStatusSelect).toHaveBeenCalledWith(job, "rejected");
    expect(onChangeStatus).not.toHaveBeenCalled();
    // menu collapses after picking
    expect(screen.queryByText("jobs.statusRejected")).not.toBeInTheDocument();
  });

  it("falls back to the modal handler when no onStatusSelect is given", () => {
    const { onChangeStatus } = renderCard();
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.changeStatus"));
    expect(onChangeStatus).toHaveBeenCalledWith(job);
  });
});

describe("JobCardBase — complete current stage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("offers the item when the job has a current stage", () => {
    const onCompleteStage = vi.fn();
    renderCard({
      job: { ...job, current_stage_id: "stage-1" },
      onCompleteStage,
    });

    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    fireEvent.click(screen.getByText("jobs.completeCurrentStage"));

    expect(onCompleteStage).toHaveBeenCalledWith({
      ...job,
      current_stage_id: "stage-1",
    });
  });

  it("hides the item when the job has no current stage", () => {
    renderCard({ onCompleteStage: vi.fn() });
    fireEvent.click(screen.getByLabelText("jobs.actionsMenu"));
    expect(
      screen.queryByText("jobs.completeCurrentStage"),
    ).not.toBeInTheDocument();
  });
});
