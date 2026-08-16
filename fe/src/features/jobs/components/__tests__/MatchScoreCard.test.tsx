import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MatchScoreCard } from "../MatchScoreCard";
import type { MatchScoreResponse } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

function makeData(overrides: Partial<MatchScoreResponse> = {}): MatchScoreResponse {
  return {
    overall_score: 82,
    categories: [],
    missing_keywords: [],
    strengths: [],
    summary: "",
    from_cache: false,
    ...overrides,
  };
}

describe("MatchScoreCard", () => {
  it("renders the overall score", () => {
    render(<MatchScoreCard data={makeData({ overall_score: 82 })} />);
    expect(screen.getByText("82%")).toBeInTheDocument();
    expect(screen.getByText("jobs.matchScore.overallScore")).toBeInTheDocument();
  });

  it("renders categories with names, scores and details", () => {
    render(
      <MatchScoreCard
        data={makeData({
          categories: [
            { name: "Skills", score: 90, details: "Strong match" },
            { name: "Experience", score: 30, details: "Junior" },
          ],
        })}
      />,
    );
    expect(screen.getByText("Skills")).toBeInTheDocument();
    expect(screen.getByText("90%")).toBeInTheDocument();
    expect(screen.getByText("Strong match")).toBeInTheDocument();
    expect(screen.getByText("Experience")).toBeInTheDocument();
    expect(screen.getByText("Junior")).toBeInTheDocument();
  });

  it("renders strengths and missing keywords as chips", () => {
    render(
      <MatchScoreCard
        data={makeData({
          strengths: ["Leadership", "React"],
          missing_keywords: ["Kubernetes"],
        })}
      />,
    );
    expect(screen.getByText("Leadership")).toBeInTheDocument();
    expect(screen.getByText("React")).toBeInTheDocument();
    expect(screen.getByText("Kubernetes")).toBeInTheDocument();
    expect(screen.getByText("jobs.matchScore.strengths")).toBeInTheDocument();
    expect(
      screen.getByText("jobs.matchScore.missingKeywords"),
    ).toBeInTheDocument();
  });

  it("renders the summary section when present", () => {
    render(
      <MatchScoreCard data={makeData({ summary: "Great fit overall." })} />,
    );
    expect(screen.getByText("Great fit overall.")).toBeInTheDocument();
    expect(screen.getByText("jobs.matchScore.summary")).toBeInTheDocument();
  });

  it("omits optional sections when data is empty", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(
      screen.queryByText("jobs.matchScore.categories"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("jobs.matchScore.strengths"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("jobs.matchScore.missingKeywords"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("jobs.matchScore.summary"),
    ).not.toBeInTheDocument();
  });

  it("applies score-tier colour classes to the overall score circle", () => {
    const { rerender } = render(
      <MatchScoreCard data={makeData({ overall_score: 85 })} />,
    );
    expect(screen.getByText("85%").parentElement?.className).toContain(
      "border-green-500",
    );

    rerender(<MatchScoreCard data={makeData({ overall_score: 50 })} />);
    expect(screen.getByText("50%").parentElement?.className).toContain(
      "border-yellow-500",
    );

    rerender(<MatchScoreCard data={makeData({ overall_score: 20 })} />);
    expect(screen.getByText("20%").parentElement?.className).toContain(
      "border-red-500",
    );
  });
});
