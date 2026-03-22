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

const makeData = (
  overrides: Partial<MatchScoreResponse> = {},
): MatchScoreResponse => ({
  overall_score: 85,
  categories: [
    { name: "Technical Skills", score: 90, details: "Strong in React" },
    { name: "Experience", score: 75, details: "3 years" },
  ],
  missing_keywords: ["Docker", "K8s"],
  strengths: ["TypeScript", "React"],
  summary: "Good overall match",
  from_cache: false,
  ...overrides,
});

describe("MatchScoreCard", () => {
  it("renders overall score title", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(
      screen.getByText("applications.matchScore.overallScore"),
    ).toBeInTheDocument();
  });

  it("renders overall score percentage", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("85%")).toBeInTheDocument();
  });

  it("renders category names and scores", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("Technical Skills")).toBeInTheDocument();
    expect(screen.getByText("90%")).toBeInTheDocument();
    expect(screen.getByText("Experience")).toBeInTheDocument();
    expect(screen.getByText("75%")).toBeInTheDocument();
  });

  it("renders category details", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("Strong in React")).toBeInTheDocument();
  });

  it("renders strengths", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("TypeScript")).toBeInTheDocument();
    expect(screen.getByText("React")).toBeInTheDocument();
  });

  it("renders missing keywords", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("Docker")).toBeInTheDocument();
    expect(screen.getByText("K8s")).toBeInTheDocument();
  });

  it("renders summary", () => {
    render(<MatchScoreCard data={makeData()} />);
    expect(screen.getByText("Good overall match")).toBeInTheDocument();
  });

  it("does not render sections when arrays are empty", () => {
    render(
      <MatchScoreCard
        data={makeData({
          categories: [],
          strengths: [],
          missing_keywords: [],
          summary: "",
        })}
      />,
    );
    expect(
      screen.queryByText("applications.matchScore.categories"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("applications.matchScore.strengths"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("applications.matchScore.missingKeywords"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("applications.matchScore.summary"),
    ).not.toBeInTheDocument();
  });

  it("uses green color for high scores", () => {
    const { container } = render(<MatchScoreCard data={makeData({ overall_score: 80 })} />);
    expect(container.querySelector(".border-green-500")).toBeTruthy();
  });

  it("uses yellow color for medium scores", () => {
    const { container } = render(<MatchScoreCard data={makeData({ overall_score: 50 })} />);
    expect(container.querySelector(".border-yellow-500")).toBeTruthy();
  });

  it("uses red color for low scores", () => {
    const { container } = render(<MatchScoreCard data={makeData({ overall_score: 20 })} />);
    expect(container.querySelector(".border-red-500")).toBeTruthy();
  });
});
