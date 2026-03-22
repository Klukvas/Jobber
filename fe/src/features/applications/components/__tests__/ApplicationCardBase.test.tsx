import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ApplicationCardBase } from "../ApplicationCardBase";
import type { ApplicationDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const makeApp = (overrides: Partial<ApplicationDTO> = {}): ApplicationDTO => ({
  id: "app-1",
  name: "Frontend Developer Application",
  status: "active",
  applied_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  ...overrides,
});

describe("ApplicationCardBase", () => {
  const defaultProps = {
    application: makeApp(),
    onTitleClick: vi.fn(),
    onAddComment: vi.fn(),
    onAddStage: vi.fn(),
    onChangeStatus: vi.fn(),
  };

  it("renders application name", () => {
    render(<ApplicationCardBase {...defaultProps} />);
    expect(screen.getByText("Frontend Developer Application")).toBeInTheDocument();
  });

  it("renders status badge", () => {
    render(<ApplicationCardBase {...defaultProps} />);
    expect(screen.getByText("applications.statusActive")).toBeInTheDocument();
  });

  it("calls onTitleClick when name is clicked", () => {
    const onTitleClick = vi.fn();
    render(
      <ApplicationCardBase {...defaultProps} onTitleClick={onTitleClick} />,
    );
    fireEvent.click(screen.getByText("Frontend Developer Application"));
    expect(onTitleClick).toHaveBeenCalledOnce();
  });

  it("renders company name when present", () => {
    const app = makeApp({
      job: {
        id: "j1",
        title: "Dev",
        company: { id: "c1", name: "Acme Corp" },
      },
    });
    render(<ApplicationCardBase {...defaultProps} application={app} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });

  it("renders job title when present", () => {
    const app = makeApp({
      job: {
        id: "j1",
        title: "Senior Engineer",
        company: { id: "c1", name: "Co" },
      },
    });
    render(<ApplicationCardBase {...defaultProps} application={app} />);
    expect(screen.getByText("Senior Engineer")).toBeInTheDocument();
  });

  it("renders actions menu button", () => {
    render(<ApplicationCardBase {...defaultProps} />);
    expect(screen.getByLabelText("applications.actionsMenu")).toBeInTheDocument();
  });

  it("shows action menu when menu button is clicked", () => {
    render(<ApplicationCardBase {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("applications.actionsMenu"));
    expect(screen.getByText("applications.addComment")).toBeInTheDocument();
    expect(screen.getByText("applications.addStage")).toBeInTheDocument();
    expect(screen.getByText("applications.changeStatus")).toBeInTheDocument();
  });

  it("calls onAddComment from menu", () => {
    const onAddComment = vi.fn();
    render(
      <ApplicationCardBase {...defaultProps} onAddComment={onAddComment} />,
    );
    fireEvent.click(screen.getByLabelText("applications.actionsMenu"));
    fireEvent.click(screen.getByText("applications.addComment"));
    expect(onAddComment).toHaveBeenCalledWith(defaultProps.application);
  });

  it("calls onAddStage from menu", () => {
    const onAddStage = vi.fn();
    render(
      <ApplicationCardBase {...defaultProps} onAddStage={onAddStage} />,
    );
    fireEvent.click(screen.getByLabelText("applications.actionsMenu"));
    fireEvent.click(screen.getByText("applications.addStage"));
    expect(onAddStage).toHaveBeenCalledWith(defaultProps.application);
  });

  it("calls onChangeStatus from menu", () => {
    const onChangeStatus = vi.fn();
    render(
      <ApplicationCardBase {...defaultProps} onChangeStatus={onChangeStatus} />,
    );
    fireEvent.click(screen.getByLabelText("applications.actionsMenu"));
    fireEvent.click(screen.getByText("applications.changeStatus"));
    expect(onChangeStatus).toHaveBeenCalledWith(defaultProps.application);
  });
});
