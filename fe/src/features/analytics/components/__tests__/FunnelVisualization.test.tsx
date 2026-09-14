import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { FunnelAnalytics } from "@/services/analyticsService";
import { FunnelVisualization } from "../FunnelVisualization";

// Counted messages are echoed back with their count, so a test can see which
// key was asked for and with what — the real forms are pinned against the real
// bundles in shared/locales/__tests__/applicationCount.test.ts.
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count === undefined ? key : `${key}:${options.count}`,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

const stages: FunnelAnalytics["stages"] = [
  {
    stage_name: "applied",
    stage_order: 1,
    count: 25,
    conversion_rate: 100,
    drop_off_rate: 0,
  },
  {
    stage_name: "in_progress",
    stage_order: 2,
    count: 10,
    conversion_rate: 40,
    drop_off_rate: 60,
    sub_stages: [
      {
        stage_name: "HR Interview",
        stage_order: 1,
        count: 8,
        conversion_rate: 0,
        drop_off_rate: 0,
      },
      {
        stage_name: "Technical Interview",
        stage_order: 2,
        count: 3,
        conversion_rate: 0,
        drop_off_rate: 0,
      },
    ],
  },
  {
    stage_name: "offer",
    stage_order: 3,
    count: 2,
    conversion_rate: 20,
    drop_off_rate: 80,
  },
];

describe("FunnelVisualization", () => {
  it("localizes phase bucket labels", () => {
    render(<FunnelVisualization data={{ stages }} isLoading={false} />);

    expect(
      screen.getByText("analytics.funnel.phaseApplied"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("analytics.funnel.phaseInProgress"),
    ).toBeInTheDocument();
    expect(screen.getByText("analytics.funnel.phaseOffer")).toBeInTheDocument();
  });

  it("renders the in-progress stage drill-down", () => {
    render(<FunnelVisualization data={{ stages }} isLoading={false} />);

    expect(
      screen.getByText("analytics.funnel.stageBreakdown"),
    ).toBeInTheDocument();
    expect(screen.getByText("HR Interview")).toBeInTheDocument();
    expect(screen.getByText("Technical Interview")).toBeInTheDocument();
  });

  it("renders the terminal rejected block with per-stage breakdown", () => {
    const data: FunnelAnalytics = {
      stages,
      rejected: {
        total: 4,
        by_stage: [
          { stage_name: "Applied", stage_order: 1, count: 3 },
          { stage_name: "Technical Interview", stage_order: 2, count: 1 },
        ],
      },
    };

    render(<FunnelVisualization data={data} isLoading={false} />);

    expect(screen.getByText("analytics.funnel.rejected")).toBeInTheDocument();
    expect(
      screen.getByText(/Applied — 3 · Technical Interview — 1/),
    ).toBeInTheDocument();
  });

  it("hides the rejected block when there are no rejections", () => {
    render(<FunnelVisualization data={{ stages }} isLoading={false} />);

    expect(
      screen.queryByText("analytics.funnel.rejected"),
    ).not.toBeInTheDocument();
  });
});

/**
 * The bar labels used to be a raw number next to an invariant noun, which read
 * "1 applications" the moment a stage held exactly one.
 */
describe("FunnelVisualization — counted totals", () => {
  it("asks for a counted message rather than gluing a number to a noun", () => {
    render(<FunnelVisualization data={{ stages }} isLoading={false} />);

    expect(
      screen.getByText("analytics.applicationsCount:25"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("analytics.applicationsCount:2"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("analytics.applications"),
    ).not.toBeInTheDocument();
  });

  it("counts the terminal rejections the same way", () => {
    render(
      <FunnelVisualization
        data={{ stages, rejected: { total: 1, by_stage: [] } }}
        isLoading={false}
      />,
    );

    expect(
      screen.getByText("analytics.applicationsCount:1"),
    ).toBeInTheDocument();
  });
});
