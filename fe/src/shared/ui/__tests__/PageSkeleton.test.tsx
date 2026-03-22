import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  ListPageSkeleton,
  DetailPageSkeleton,
  EditorPageSkeleton,
  CoverLetterListSkeleton,
  StageTemplateListSkeleton,
  SettingsPageSkeleton,
} from "../PageSkeleton";

describe("ListPageSkeleton", () => {
  it("renders with role=status", () => {
    render(<ListPageSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders default 6 card placeholders", () => {
    const { container } = render(<ListPageSkeleton />);
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(cards.length).toBe(6);
  });

  it("renders custom card count", () => {
    const { container } = render(<ListPageSkeleton cards={3} />);
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(cards.length).toBe(3);
  });
});

describe("DetailPageSkeleton", () => {
  it("renders with role=status", () => {
    render(<DetailPageSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders content blocks", () => {
    const { container } = render(<DetailPageSkeleton />);
    const blocks = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(blocks.length).toBe(3);
  });
});

describe("EditorPageSkeleton", () => {
  it("renders with role=status", () => {
    render(<EditorPageSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });
});

describe("CoverLetterListSkeleton", () => {
  it("renders with role=status", () => {
    render(<CoverLetterListSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders default 6 cards", () => {
    const { container } = render(<CoverLetterListSkeleton />);
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(cards.length).toBe(6);
  });

  it("renders custom card count", () => {
    const { container } = render(<CoverLetterListSkeleton cards={2} />);
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(cards.length).toBe(2);
  });
});

describe("StageTemplateListSkeleton", () => {
  it("renders with role=status", () => {
    render(<StageTemplateListSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders default 5 rows", () => {
    const { container } = render(<StageTemplateListSkeleton />);
    // Stage template rows are inside the last .space-y-3
    const rows = container.querySelectorAll(".space-y-3 > .rounded-lg.border.bg-card");
    expect(rows.length).toBe(5);
  });
});

describe("SettingsPageSkeleton", () => {
  it("renders with role=status", () => {
    render(<SettingsPageSkeleton />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders 4 settings cards", () => {
    const { container } = render(<SettingsPageSkeleton />);
    const cards = container.querySelectorAll(".rounded-lg.border.bg-card");
    expect(cards.length).toBe(4);
  });
});
