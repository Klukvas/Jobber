import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ApplicationKanbanCard } from "../ApplicationKanbanCard";
import type { ApplicationDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("@dnd-kit/core", () => ({
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
}));

vi.mock("../ApplicationCardBase", () => ({
  ApplicationCardBase: ({
    application,
  }: {
    application: ApplicationDTO;
  }) => <div data-testid="card-base">{application.name}</div>,
}));

const makeApp = (): ApplicationDTO => ({
  id: "app-1",
  name: "Test App",
  status: "active",
  applied_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
});

describe("ApplicationKanbanCard", () => {
  it("renders application card base", () => {
    render(
      <ApplicationKanbanCard
        application={makeApp()}
        onAddComment={vi.fn()}
        onAddStage={vi.fn()}
        onChangeStatus={vi.fn()}
      />,
    );
    expect(screen.getByTestId("card-base")).toBeInTheDocument();
    expect(screen.getByText("Test App")).toBeInTheDocument();
  });
});
