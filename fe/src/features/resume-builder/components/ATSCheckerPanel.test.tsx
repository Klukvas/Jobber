import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ATSCheckerPanel } from "./ATSCheckerPanel";

const mockMutate = vi.fn();

const mockATSCheckRef = {
  current: {
    mutate: mockMutate,
    isPending: false,
    isError: false,
    data: null as null | {
      score: number;
      issues: Array<{
        severity: "critical" | "warning" | "info";
        description: string;
      }>;
      suggestions: string[];
      keywords_found: string[];
    },
  },
};

const mockResumeRef = {
  current: null as { id: string } | null,
};

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ resume: mockResumeRef.current }),
}));

vi.mock("../hooks/useATSCheck", () => ({
  useATSCheck: () => mockATSCheckRef.current,
}));

describe("ATSCheckerPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockResumeRef.current = { id: "resume-1" };
    mockATSCheckRef.current = {
      mutate: mockMutate,
      isPending: false,
      isError: false,
      data: null,
    };
  });

  it("renders the title", () => {
    render(<ATSCheckerPanel />);
    expect(screen.getByText("resumeBuilder.ats.title")).toBeInTheDocument();
  });

  it("renders the check button", () => {
    render(<ATSCheckerPanel />);
    expect(screen.getByText("resumeBuilder.ats.check")).toBeInTheDocument();
  });

  it("disables check button when no resume is loaded", () => {
    mockResumeRef.current = null;
    render(<ATSCheckerPanel />);
    const btn = screen.getByRole("button");
    expect(btn).toBeDisabled();
  });

  it("calls mutate with resume id when check button is clicked", async () => {
    const user = userEvent.setup();
    render(<ATSCheckerPanel />);

    await user.click(screen.getByText("resumeBuilder.ats.check"));
    expect(mockMutate).toHaveBeenCalledWith("resume-1");
  });

  it("shows loading state when isPending is true", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      isPending: true,
    };
    render(<ATSCheckerPanel />);
    expect(
      screen.getByText("resumeBuilder.ats.checking"),
    ).toBeInTheDocument();
  });

  it("shows error text when isError is true", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      isError: true,
    };
    render(<ATSCheckerPanel />);
    expect(screen.getByText("common.error")).toBeInTheDocument();
  });

  it("renders the score when result data is available", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      data: {
        score: 85,
        issues: [],
        suggestions: [],
        keywords_found: [],
      },
    };
    render(<ATSCheckerPanel />);
    expect(screen.getByText("85")).toBeInTheDocument();
    expect(screen.getByText("resumeBuilder.ats.score")).toBeInTheDocument();
  });

  it("renders issues when present", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      data: {
        score: 60,
        issues: [
          { severity: "critical", description: "Missing email" },
          { severity: "warning", description: "Short summary" },
        ],
        suggestions: [],
        keywords_found: [],
      },
    };
    render(<ATSCheckerPanel />);
    expect(screen.getByText("Missing email")).toBeInTheDocument();
    expect(screen.getByText("Short summary")).toBeInTheDocument();
  });

  it("shows no-issues text when result has no issues", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      data: {
        score: 95,
        issues: [],
        suggestions: [],
        keywords_found: [],
      },
    };
    render(<ATSCheckerPanel />);
    expect(
      screen.getByText("resumeBuilder.ats.noIssues"),
    ).toBeInTheDocument();
  });

  it("renders suggestions when present", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      data: {
        score: 70,
        issues: [],
        suggestions: ["Add more keywords", "Use action verbs"],
        keywords_found: [],
      },
    };
    render(<ATSCheckerPanel />);
    expect(screen.getByText("Add more keywords")).toBeInTheDocument();
    expect(screen.getByText("Use action verbs")).toBeInTheDocument();
  });

  it("renders keywords when present", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      data: {
        score: 80,
        issues: [],
        suggestions: [],
        keywords_found: ["React", "TypeScript", "Node.js"],
      },
    };
    render(<ATSCheckerPanel />);
    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getByText("Node.js")).toBeInTheDocument();
  });
});
