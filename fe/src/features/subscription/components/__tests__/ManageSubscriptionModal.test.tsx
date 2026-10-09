import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";
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

const api = vi.hoisted(() => ({
  changePlan: vi.fn(),
  cancelSubscription: vi.fn(),
  createPortalSession: vi.fn(),
}));

vi.mock("@/services/subscriptionService", () => ({
  subscriptionService: api,
}));

const notifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/lib/notifications", () => notifications);

// `@/shared/ui/Dialog` is deliberately not mocked: it is what gives this modal
// its accessible name, and a stub would quietly take that back.
//
// A real query client rather than a hand-rolled useMutation stub: these flows
// live in onSuccess/onError, which a stub that just calls mutationFn never runs.
function renderModal(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("ManageSubscriptionModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.changePlan.mockResolvedValue(undefined);
    api.cancelSubscription.mockResolvedValue(undefined);
    api.createPortalSession.mockResolvedValue({
      url: "https://creem.io/my-orders/abc#/subscriptions",
    });
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
    const { container } = renderModal(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when subscription is null", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      subscription: null as never,
    };
    const { container } = renderModal(
      <ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the modal title when open with a paid plan", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.manage.title"),
    ).toBeInTheDocument();
  });

  it("renders the current plan label for pro", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.currentPlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.proPlan"),
    ).toBeInTheDocument();
  });

  /**
   * The prices used to be a second, English-only copy of the pricing table:
   * "$7/mo" hard-coded here while the pricing modal read
   * `settings.subscription.pricing.proPrice` — which RU and UK translate as
   * "$7/мес" and "$7/міс". A Russian-speaking subscriber saw "/mo" in this
   * modal and "/мес" one screen away, and a price change had two places to
   * land in.
   */
  it("quotes prices from the localised pricing copy, not a second English table", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    expect(
      screen.getByText("settings.subscription.pricing.proPrice"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.pricing.enterprisePrice"),
    ).toBeInTheDocument();
    expect(screen.queryByText(/\$\d+\/mo/)).not.toBeInTheDocument();
  });

  it("quotes the free price from the same source when a plan lapses to free", () => {
    mockSubscriptionRef.current = {
      ...mockSubscriptionRef.current,
      plan: "enterprise",
      subscription: {
        ...mockSubscriptionRef.current.subscription,
        plan: "enterprise",
      },
    };
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    expect(
      screen.getByText("settings.subscription.pricing.enterprisePrice"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.pricing.proPrice"),
    ).toBeInTheDocument();
  });

  it("shows upgrade to enterprise option when on pro plan", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.enterprisePlan"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.manage.switchToEnterprise"),
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
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.manage.switchToPro"),
    ).toBeInTheDocument();
  });

  it("renders cancel subscription link when not cancelled", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("settings.subscription.manage.cancelSubscription"),
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
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.queryByText("settings.subscription.manage.cancelSubscription"),
    ).not.toBeInTheDocument();
  });

  it("renders nothing when open is false", () => {
    const { container } = renderModal(
      <ManageSubscriptionModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });
});

describe("ManageSubscriptionModal — billing portal", () => {
  // Invoices, receipts, the payment method and refund requests all live on the
  // provider's side. Without this button the refund policy promises a route the
  // app never offered.
  let assignSpy: ReturnType<typeof vi.fn>;
  let originalLocation: Location;

  beforeEach(() => {
    vi.clearAllMocks();
    api.createPortalSession.mockResolvedValue({
      url: "https://creem.io/my-orders/abc#/subscriptions",
    });
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

    assignSpy = vi.fn();
    originalLocation = window.location;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { ...originalLocation, assign: assignSpy },
    });
  });

  afterEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
  });

  it("offers the portal to a paying subscriber", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    expect(
      screen.getByText("settings.subscription.manage.billingPortal"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("settings.subscription.manage.openBillingPortal"),
    ).toBeInTheDocument();
  });

  it("navigates to the URL the backend returned, in the same tab", async () => {
    // window.open after an await is severed from the click and gets blocked as
    // a popup, so the portal must be reached by navigating this tab.
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    fireEvent.click(
      screen.getByText("settings.subscription.manage.openBillingPortal"),
    );

    await waitFor(() =>
      expect(assignSpy).toHaveBeenCalledExactlyOnceWith(
        "https://creem.io/my-orders/abc#/subscriptions",
      ),
    );
    expect(api.createPortalSession).toHaveBeenCalledOnce();
  });

  it.each([
    ["a javascript: URL", "javascript:alert(1)"],
    ["an http URL", "http://creem.io/my-orders/abc"],
    ["a URL with userinfo", "https://creem.io@evil.com/x"],
  ])("refuses to navigate to %s from the backend", async (_label, url) => {
    api.createPortalSession.mockResolvedValue({ url });

    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    fireEvent.click(
      screen.getByText("settings.subscription.manage.openBillingPortal"),
    );

    await waitFor(() =>
      expect(notifications.showErrorNotification).toHaveBeenCalledWith(
        "settings.subscription.manage.portalError",
      ),
    );
    expect(assignSpy).not.toHaveBeenCalled();
  });

  it("reports a failure instead of navigating nowhere", async () => {
    api.createPortalSession.mockRejectedValue(new Error("portal unavailable"));

    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    fireEvent.click(
      screen.getByText("settings.subscription.manage.openBillingPortal"),
    );

    await waitFor(() =>
      expect(notifications.showErrorNotification).toHaveBeenCalledWith(
        "settings.subscription.manage.portalError",
      ),
    );
    expect(assignSpy).not.toHaveBeenCalled();
  });

  it("blocks the other actions while the portal session is being created", async () => {
    let releasePortal: (value: { url: string }) => void = () => {};
    api.createPortalSession.mockReturnValue(
      new Promise<{ url: string }>((resolve) => {
        releasePortal = resolve;
      }),
    );

    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    fireEvent.click(
      screen.getByText("settings.subscription.manage.openBillingPortal"),
    );

    await waitFor(() =>
      expect(
        screen
          .getByText("settings.subscription.manage.switchToEnterprise")
          .closest("button"),
      ).toBeDisabled(),
    );

    releasePortal({ url: "https://creem.io/my-orders/abc" });
  });
});

/**
 * The one modal in the app with no heading component at all, so `role="dialog"`
 * was announced as an unnamed dialog.
 */
describe("ManageSubscriptionModal — accessible name", () => {
  it("is named by its own heading", () => {
    renderModal(<ManageSubscriptionModal open={true} onOpenChange={vi.fn()} />);

    const labelledBy = screen
      .getByRole("dialog")
      .getAttribute("aria-labelledby");

    expect(labelledBy).toBeTruthy();
    expect(document.getElementById(labelledBy ?? "")?.textContent).toBe(
      "settings.subscription.manage.title",
    );
  });
});
