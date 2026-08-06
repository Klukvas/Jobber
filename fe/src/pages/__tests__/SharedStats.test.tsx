import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";

const mockGetPublic = vi.hoisted(() => vi.fn());

vi.mock("@/services/sharingService", () => ({
  sharingService: { getPublic: mockGetPublic },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ token: "tok-1" }),
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

import SharedStats from "../SharedStats";

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <SharedStats />
    </QueryClientProvider>,
  );
}

describe("SharedStats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the shared snapshot", async () => {
    mockGetPublic.mockResolvedValue({
      snapshot: {
        schema_version: 1,
        generated_at: "2026-08-06T12:00:00Z",
        overview: {
          total_applications: 127,
          active_applications: 40,
          closed_applications: 87,
          response_rate: 18.5,
          avg_days_to_first_response: 6.2,
        },
        funnel: [
          {
            stage_name: "Applied",
            stage_order: 1,
            count: 127,
            conversion_rate: 100,
            drop_off_rate: 0,
          },
        ],
      },
      created_at: "2026-08-06T12:00:00Z",
    });

    renderPage();

    await waitFor(() => {
      // appears in the overview card and on the funnel bar
      expect(screen.getAllByText("127").length).toBeGreaterThan(0);
    });
    expect(mockGetPublic).toHaveBeenCalledWith("tok-1");
    expect(screen.getByText("Applied")).toBeInTheDocument();
    expect(screen.getByText("18.5%")).toBeInTheDocument();
    expect(screen.getByText("sharing.public.ctaTitle")).toBeInTheDocument();
  });

  it("shows the not-found state for a revoked link", async () => {
    mockGetPublic.mockRejectedValue(
      new ApiError("Share not found", "SHARE_NOT_FOUND", 404),
    );

    renderPage();

    await waitFor(() => {
      expect(
        screen.getByText("sharing.public.notFoundTitle"),
      ).toBeInTheDocument();
    });
    // CTA must survive the not-found state — that's the growth loop
    expect(screen.getByText("sharing.public.ctaTitle")).toBeInTheDocument();
  });
});
