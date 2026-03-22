import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ManageSubscriptionModal } from "../ManageSubscriptionModal";

const mockSubscriptionRef = vi.hoisted(() => ({
  current: {
    plan: "pro" as "free" | "pro" | "enterprise",
    subscription: {
      plan: "pro" as "free" | "pro" | "enterprise",
      status: "active" as const,
      current_period_end: "2024-12-31T00:00:00Z",
      cancel_at: null as string | null,
      limits: {
        max_jobs: -1,
        max_resumes: -1,
        max_applications: -1,
        max_ai_requests: 50,
        max_job_parses: -1,
        max_resume_builders: -1,
        max_cover_letters: -1,
      },
      usage: {
        jobs: 0,
        resumes: 0,
        applications: 0,
        ai_requests: 0,
        job_parses: 0,
        resume_builders: 0,
        cover_letters: 0,
      },
    },
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params) {
        return Object.entries(params).reduce(
          (acc, [k, v]) => acc.replace(`{{${k}}}`, String(v)),
          key,
        );
      }
      return key;
    },
    i18n: { language: "en" },
  }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => mockSubscriptionRef.current,
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: {
    changePlan: vi.fn().mockResolvedValue({}),
    cancelSubscription: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: ({ mutationFn }: { mutationFn: unknown }) => ({
    mutate: mutationFn,
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/shared/ui/Dialog", () => ({
  Dialog: ({
    open,
    children,
  }: {
    open: boolean;
    children: React.ReactNode;
  }) => (open ? <div data-testid="dialog">{children}</div> : null),
}));

describe("ManageSubscriptionModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSubscriptionRef.current = {
      plan: "pro",
      subscription: {
        plan: "pro",
        status: "active",
        current_period_end: "2024-12-31T00:00:00Z",
        cancel_at: null,
        limits: {
          max_jobs: -1,
          max_resumes: -1,
          max_applications: -1,
          max_ai_requests: 50,
          max_job_parses: -1,
          max_resume_builders: -1,
          max_cover_letters: -1,
        },
        usage: {
          jobs: 0,
          resumes: 0,
          applications: 0,
          ai_requests: 0,
          job_parses: 0,
          resume_builders: 0,
          cover_letters: 0,
        },
      },
    };
  });

  it("renders nothing when plan is free", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      plan: "free",
    };
    const { container } = render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when subscription is null", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      subscription: null as never,
    };
    const { container } = render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the modal title when open with a paid plan", () => {
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.getByText("settings.subscription.manage.title"),
    ).toBeInTheDocument();
  });

  it("renders the current plan label for pro", () => {
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.getByText("settings.subscription.currentPlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.proPlan"),
    ).toBeInTheDocument();
  });

  it("shows upgrade to enterprise option when on pro plan", () => {
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.getByText("settings.subscription.enterprisePlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "settings.subscription.manage.switchToEnterprise",
      ),
    ).toBeInTheDocument();
  });

  it("shows downgrade to pro option when on enterprise plan", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      plan: "enterprise",
      subscription: {
        ...mockSubscriptionRef.current.subscription,
        plan: "enterprise",
      },
    };
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.getByText("settings.subscription.manage.switchToPro"),
    ).toBeInTheDocument();
  });

  it("renders cancel subscription link when not cancelled", () => {
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.getByText(
        "settings.subscription.manage.cancelSubscription",
      ),
    ).toBeInTheDocument();
  });

  it("does not render cancel link when already cancelled", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      subscription: {
        ...mockSubscriptionRef.current.subscription,
        cancel_at: "2025-01-31T00:00:00Z",
      },
    };
    render(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(
      screen.queryByText(
        "settings.subscription.manage.cancelSubscription",
      ),
    ).not.toBeInTheDocument();
  });

  it("renders nothing when open is false", () => {
    const { container } = render(
      <ManageSubscriptionModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });
});
