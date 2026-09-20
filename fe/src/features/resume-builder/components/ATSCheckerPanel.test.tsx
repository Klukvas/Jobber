import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ATSCheckerPanel } from "./ATSCheckerPanel";
import { ApiError } from "@/services/api";

const mockMutate = vi.fn();

const mockATSCheckRef = {
  current: {
    mutate: mockMutate,
    isPending: false,
    isError: false,
    error: null as unknown,
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

// Interpolation values are appended to the key so a test can assert not just
// which message is shown but the number it was given.
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options && "count" in options ? `${key}:${options.count}` : key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    limits: {
      max_jobs: 25,
      max_resumes: 1,
      max_ai_requests: 1,
      max_job_parses: 5,
      max_resume_builders: 1,
      max_cover_letters: 0,
    },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ resume: mockResumeRef.current }),
}));

vi.mock("../hooks/useATSCheck", () => ({
  useATSCheck: () => mockATSCheckRef.current,
}));

// The upgrade CTA is behind the payments flag, which is read from the
// environment at import time. Reading the real one made this file pass locally,
// where .env turns payments on, and fail in CI, which has no .env at all. The
// flag is pinned here instead, and exercised both ways below.
const featuresMock = vi.hoisted(() => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/shared/lib/features", () => featuresMock);

describe("ATSCheckerPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    featuresMock.FEATURES.PAYMENTS = true;
    mockResumeRef.current = { id: "resume-1" };
    mockATSCheckRef.current = {
      mutate: mockMutate,
      isPending: false,
      isError: false,
      error: null,
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
    // The success callback is how the panel keeps the last good report around.
    expect(mockMutate).toHaveBeenCalledWith(
      "resume-1",
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
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
    expect(screen.getByText("resumeBuilder.ats.error")).toBeInTheDocument();
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

// Running out of the free monthly AI allowance is a quota, not a fault: the
// panel has to say which limit was hit, when it returns, and offer the upgrade
// — and it must not throw away the report already on screen.
describe("ATSCheckerPanel — monthly AI limit", () => {
  const RESULT = {
    score: 72,
    issues: [],
    suggestions: [],
    keywords_found: ["golang"],
  };

  beforeEach(() => {
    vi.clearAllMocks();
    featuresMock.FEATURES.PAYMENTS = true;
    mockResumeRef.current = { id: "resume-1" };
    mockATSCheckRef.current = {
      mutate: mockMutate,
      isPending: false,
      isError: false,
      error: null,
      data: null,
    };
  });

  function quotaRejected() {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      isError: true,
      error: new ApiError("limit", "PLAN_LIMIT_REACHED", 403),
      data: null,
    };
  }

  // The old copy said AI match scoring was "not available on your plan" — next
  // to a reset date, and about a feature this panel does not even run. The
  // message names the shared monthly allowance and how big it is.
  it("names the monthly AI allowance instead of showing a bare error", () => {
    quotaRejected();
    render(<ATSCheckerPanel />);

    expect(
      screen.getByText("settings.subscription.limitReachedAIMonthly:1"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("resumeBuilder.ats.error"),
    ).not.toBeInTheDocument();
  });

  it("says when the allowance comes back", () => {
    quotaRejected();
    render(<ATSCheckerPanel />);

    expect(
      screen.getByText(/settings\.subscription\.resetsOn/),
    ).toBeInTheDocument();
  });

  it("offers a way to upgrade", () => {
    quotaRejected();
    render(<ATSCheckerPanel />);

    expect(
      screen.getByText("settings.subscription.upgradeForMore"),
    ).toBeInTheDocument();
  });

  it("offers no upgrade when this deployment has no payments", () => {
    featuresMock.FEATURES.PAYMENTS = false;
    quotaRejected();
    render(<ATSCheckerPanel />);

    expect(
      screen.queryByText("settings.subscription.upgradeForMore"),
    ).not.toBeInTheDocument();
  });

  it("keeps the previous report visible when a re-check is refused", async () => {
    const user = userEvent.setup();

    // First run succeeds and the panel records the result.
    mockMutate.mockImplementation(
      (_id: string, opts?: { onSuccess?: (d: typeof RESULT) => void }) =>
        opts?.onSuccess?.(RESULT),
    );
    const { rerender } = render(<ATSCheckerPanel />);
    await user.click(screen.getByText("resumeBuilder.ats.check"));

    // Second run is refused: react-query clears `data`, but the last good
    // report must survive.
    quotaRejected();
    rerender(<ATSCheckerPanel />);

    expect(screen.getByText("72")).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.limitReachedAIMonthly:1"),
    ).toBeInTheDocument();
  });

  it("still shows a plain error for a non-quota failure", () => {
    mockATSCheckRef.current = {
      ...mockATSCheckRef.current,
      isError: true,
      error: new ApiError("boom", "INTERNAL_ERROR", 500),
    };
    render(<ATSCheckerPanel />);

    expect(screen.getByText("resumeBuilder.ats.error")).toBeInTheDocument();
    expect(
      screen.queryByText("settings.subscription.limitReachedAIMonthly:1"),
    ).not.toBeInTheDocument();
  });
});
