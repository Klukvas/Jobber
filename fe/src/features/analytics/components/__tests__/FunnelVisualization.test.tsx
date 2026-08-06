import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import type { FunnelAnalytics } from "@/services/analyticsService";
import { FunnelVisualization } from "../FunnelVisualization";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

const stages: FunnelAnalytics["stages"] = [
  {
    stage_name: "Applied",
    stage_order: 1,
    count: 25,
    conversion_rate: 100,
    drop_off_rate: 0,
  },
  {
    stage_name: "Technical Interview",
    stage_order: 2,
    count: 10,
    conversion_rate: 40,
    drop_off_rate: 60,
  },
];

describe("FunnelVisualization", () => {
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
