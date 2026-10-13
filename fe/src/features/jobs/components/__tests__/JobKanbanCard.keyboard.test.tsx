import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { DndContext, KeyboardSensor, useSensor, useSensors } from "@dnd-kit/core";
import { JobKanbanCard } from "../JobKanbanCard";
import type { JobDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

// Real dnd-kit on purpose: the bug lives in how its KeyboardSensor reacts to
// keydown events bubbling up from the card's own buttons.
const onDragStart = vi.fn();

function Harness({ onAddComment }: { onAddComment: (job: JobDTO) => void }) {
  const sensors = useSensors(useSensor(KeyboardSensor));
  return (
    <MemoryRouter>
      <DndContext sensors={sensors} onDragStart={onDragStart}>
        <JobKanbanCard
          job={job}
          onAddComment={onAddComment}
          onAddStage={vi.fn()}
          onDelete={vi.fn()}
        />
      </DndContext>
    </MemoryRouter>
  );
}

const job: JobDTO = {
  id: "job-1",
  title: "Backend Engineer",
  is_archived: false,
  is_favorite: false,
  company_name: "Acme",
  current_stage_template_id: "tpl-1",
  current_stage_name: "Screening",
  last_activity_at: "2026-08-01T00:00:00Z",
  created_at: "2026-07-01T00:00:00Z",
  updated_at: "2026-08-01T00:00:00Z",
};

describe("JobKanbanCard — keyboard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("Enter on the actions button opens the menu instead of starting a drag", async () => {
    const user = userEvent.setup();
    render(<Harness onAddComment={vi.fn()} />);

    const toggle = screen.getByRole("button", { name: "jobs.actionsMenu" });
    toggle.focus();
    await user.keyboard("{Enter}");

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(onDragStart).not.toHaveBeenCalled();
  });

  it("Space on the actions button opens the menu instead of starting a drag", async () => {
    const user = userEvent.setup();
    render(<Harness onAddComment={vi.fn()} />);

    const toggle = screen.getByRole("button", { name: "jobs.actionsMenu" });
    toggle.focus();
    await user.keyboard(" ");

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(onDragStart).not.toHaveBeenCalled();
  });

  it("Enter on a menu item activates it, and Escape closes the menu", async () => {
    const user = userEvent.setup();
    const onAddComment = vi.fn();
    render(<Harness onAddComment={onAddComment} />);

    const toggle = screen.getByRole("button", { name: "jobs.actionsMenu" });
    await user.click(toggle);
    // Focus moves to the first item when the menu opens.
    await user.keyboard("{Enter}");
    expect(onAddComment).toHaveBeenCalledWith(job);
    expect(onDragStart).not.toHaveBeenCalled();

    await user.click(toggle);
    expect(screen.getByRole("menu")).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "false");
  });

  it("the focused card itself can still be picked up with the keyboard", async () => {
    const user = userEvent.setup();
    render(<Harness onAddComment={vi.fn()} />);

    // The card wrapper is the draggable's focusable element.
    const card = document.querySelector<HTMLElement>(
      '[aria-roledescription="draggable"]',
    );
    expect(card).not.toBeNull();
    card!.focus();
    await user.keyboard("{Enter}");

    expect(onDragStart).toHaveBeenCalledTimes(1);
  });
});
