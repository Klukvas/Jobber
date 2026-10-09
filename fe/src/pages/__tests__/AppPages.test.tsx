import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import Companies from "../Companies";
import Jobs from "../Jobs";
import JobDetail from "../JobDetail";
import Analytics from "../Analytics";
import Settings from "../Settings";
import StageTemplates from "../StageTemplates";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: {
      language: "en",
      changeLanguage: vi.fn(),
      getFixedT: () => (key: string) => key,
    },
  }),
}));

const mockSearchParams = vi.hoisted(() => ({
  value: new URLSearchParams(),
  set: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ id: "test-id" }),
  useLocation: () => ({ pathname: "/", search: "" }),
  useSearchParams: () => [mockSearchParams.value, mockSearchParams.set],
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  Navigate: () => null,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

const mockNotify = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
  showInfoNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotify);

vi.mock("@/shared/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

/** Mutable so a test can switch payments on for the upgrade call to action. */
const mockFeatures = vi.hoisted(() => ({
  value: {
    GOOGLE_CALENDAR: false,
    SENTRY: false,
    EMAIL_NOTIFICATIONS: false,
    PAYMENTS: false,
  },
}));

vi.mock("@/shared/lib/features", () => ({ FEATURES: mockFeatures.value }));

const mockInvalidateQueries = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  keepPreviousData: (prev: unknown) => prev,
  useQuery: () => ({
    data: null,
    isLoading: false,
    isFetching: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
  }),
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
    cancelQueries: vi.fn(),
    setQueryData: vi.fn(),
    getQueryData: vi.fn(),
  }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      user: null,
      isAuthenticated: false,
      setAuth: vi.fn(),
      clearAuth: vi.fn(),
    }),
}));

vi.mock("@/stores/themeStore", () => {
  const state = {
    theme: "light",
    setTheme: vi.fn(),
    toggleTheme: vi.fn(),
  };
  return {
    useThemeStore: (selector?: (s: Record<string, unknown>) => unknown) =>
      selector ? selector(state) : state,
  };
});

const mockSubscription = vi.hoisted(() => ({
  value: {
    subscription: null as unknown,
    isPro: false,
    isEnterprise: false,
    nextPlan: null as string | null,
    canCreate: () => true,
    usage: {
      jobs: 2,
      resumes: 1,
      applications: 0,
      ai_requests: 1,
      job_parses: 0,
      resume_builders: 1,
      cover_letters: 0,
    },
    limits: {
      max_jobs: 10,
      max_resumes: 3,
      max_applications: 20,
      max_ai_requests: 5,
      max_job_parses: 5,
      max_resume_builders: 3,
      max_cover_letters: 0,
    },
  },
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => mockSubscription.value,
}));

vi.mock("@/services/applicationsService", () => ({
  applicationsService: {
    list: vi.fn(),
    getById: vi.fn(),
    listStages: vi.fn(),
  },
}));

vi.mock("@/services/companiesService", () => ({
  companiesService: {
    list: vi.fn(),
    toggleFavorite: vi.fn(),
  },
}));

vi.mock("@/services/jobsService", () => ({
  jobsService: {
    list: vi.fn(),
    getById: vi.fn(),
    archive: vi.fn(),
    update: vi.fn(),
    toggleFavorite: vi.fn(),
  },
}));

vi.mock("@/services/resumesService", () => ({
  resumesService: { list: vi.fn() },
}));

vi.mock("@/services/analyticsService", () => ({
  analyticsService: {
    getOverview: vi.fn(),
    getFunnel: vi.fn(),
    getStageTime: vi.fn(),
    getResumeEffectiveness: vi.fn(),
    getSourceAnalytics: vi.fn(),
  },
}));

vi.mock("@/services/commentsService", () => ({
  commentsService: { create: vi.fn() },
}));

vi.mock("@/services/matchScoreService", () => ({
  matchScoreService: { checkMatch: vi.fn() },
}));

vi.mock("@/services/authService", () => ({
  authService: { logout: vi.fn() },
}));

vi.mock("@/services/calendarService", () => ({
  calendarService: {
    getStatus: vi.fn(),
    getAuthURL: vi.fn(),
    disconnect: vi.fn(),
  },
}));

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  ApiError: class ApiError extends Error {
    code: string;
    status: number;
    constructor(message: string, code: string, status: number) {
      super(message);
      this.code = code;
      this.status = status;
    }
  },
}));

vi.mock("@/features/applications/modals/CreateApplicationModal", () => ({
  CreateApplicationModal: () => null,
}));

vi.mock("@/features/applications/modals/AddCommentModal", () => ({
  AddCommentModal: () => null,
}));

vi.mock("@/features/applications/modals/AddStageModal", () => ({
  AddStageModal: () => null,
}));

vi.mock("@/features/applications/modals/UpdateApplicationStatusModal", () => ({
  UpdateApplicationStatusModal: () => null,
}));

vi.mock("@/features/applications/components/ApplicationKanbanBoard", () => ({
  ApplicationKanbanBoard: () => null,
  APPLICATIONS_KANBAN_QUERY_KEY: ["applications-kanban"],
}));

vi.mock("@/features/applications/components/Timeline", () => ({
  Timeline: () => null,
}));

vi.mock("@/features/applications/components/MatchScoreCard", () => ({
  MatchScoreCard: () => null,
}));

vi.mock("@/features/companies/modals/CreateCompanyModal", () => ({
  CreateCompanyModal: () => null,
}));

vi.mock("@/features/companies/modals/DeleteCompanyDialog", () => ({
  DeleteCompanyDialog: () => null,
}));

vi.mock("@/features/jobs/modals/CreateJobModal", () => ({
  CreateJobModal: () => null,
}));

vi.mock("@/features/jobs/components/CompanySelectWithQuickAdd", () => ({
  CompanySelectWithQuickAdd: () => null,
}));

vi.mock("@/features/stages/modals/CreateStageTemplateModal", () => ({
  CreateStageTemplateModal: () => null,
}));

vi.mock("@/features/stages/modals/EditStageTemplateModal", () => ({
  EditStageTemplateModal: () => null,
}));

vi.mock("@/features/subscription/components/PricingModal", () => ({
  PricingModal: () => null,
}));

vi.mock("@/features/subscription/components/ManageSubscriptionModal", () => ({
  ManageSubscriptionModal: () => null,
}));

vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboarding: () => ({ restart: vi.fn() }),
}));

describe("Companies", () => {
  it("renders page title", () => {
    render(<Companies />);
    expect(screen.getByText("companies.title")).toBeInTheDocument();
  });

  it("shows empty state when no data", () => {
    render(<Companies />);
    expect(screen.getByText("companies.noCompanies")).toBeInTheDocument();
  });

  it("shows create button", () => {
    render(<Companies />);
    expect(screen.getAllByText("companies.create").length).toBeGreaterThan(0);
  });
});

describe("Jobs", () => {
  it("renders page title", () => {
    render(<Jobs />);
    expect(screen.getByText("jobs.title")).toBeInTheDocument();
  });

  it("shows empty state when no data", () => {
    render(<Jobs />);
    expect(screen.getByText("jobs.emptyTitle")).toBeInTheDocument();
  });

  it("shows create button", () => {
    render(<Jobs />);
    expect(screen.getAllByText("jobs.create").length).toBeGreaterThan(0);
  });
});

describe("JobDetail", () => {
  it("renders without crash and shows back button", () => {
    render(<JobDetail />);
    expect(screen.getAllByText("jobs.backToJobs").length).toBeGreaterThan(0);
  });

  it("shows error state when no job data", () => {
    render(<JobDetail />);
    expect(screen.getByText("jobs.notFound")).toBeInTheDocument();
  });

  // Measured 137x40 on a phone. It is the shared Button at its default size,
  // so this asserts the call site actually gets the size scale's phone floor
  // rather than a hand-rolled element that would miss it.
  it("gives the back button a 44px minimum on phones", () => {
    render(<JobDetail />);
    const back = screen.getAllByText("jobs.backToJobs")[0].closest("button");

    expect(back?.className).toContain("max-sm:h-11");
  });
});

describe("Analytics", () => {
  it("renders page title", () => {
    render(<Analytics />);
    expect(screen.getByText("analytics.title")).toBeInTheDocument();
  });

  it("shows no data state when overview returns null", () => {
    render(<Analytics />);
    // With null data and not loading, it renders the page title at minimum
    expect(screen.getByText("analytics.title")).toBeInTheDocument();
  });
});

describe("Settings", () => {
  it("renders page title", () => {
    render(<Settings />);
    expect(screen.getByText("settings.title")).toBeInTheDocument();
  });

  it("shows theme section", () => {
    render(<Settings />);
    expect(screen.getByText("settings.theme")).toBeInTheDocument();
  });

  it("shows language section", () => {
    render(<Settings />);
    expect(screen.getByText("settings.language")).toBeInTheDocument();
  });

  it("shows subscription section", () => {
    render(<Settings />);
    expect(screen.getByText("settings.subscription.title")).toBeInTheDocument();
  });

  it("shows account section with logout", () => {
    render(<Settings />);
    expect(screen.getByText("settings.account")).toBeInTheDocument();
    expect(screen.getByText("auth.logout")).toBeInTheDocument();
  });

  // The usage grid used to list only jobs, resumes and AI requests, so two
  // allowances that can block the customer were invisible.
  describe("plan usage", () => {
    it("lists every allowance the plan limits", () => {
      render(<Settings />);

      expect(
        screen.getByText("settings.subscription.jobs"),
      ).toBeInTheDocument();
      expect(
        screen.getByText("settings.subscription.resumes"),
      ).toBeInTheDocument();
      expect(
        screen.getByText("settings.subscription.aiRequests"),
      ).toBeInTheDocument();
      expect(
        screen.getByText("settings.subscription.resumeBuilders"),
      ).toBeInTheDocument();
      expect(
        screen.getByText("settings.subscription.coverLetters"),
      ).toBeInTheDocument();
    });

    it("marks a zero allowance as not included rather than as 0 of 0", () => {
      render(<Settings />);
      expect(
        screen.getByText("settings.subscription.notIncluded"),
      ).toBeInTheDocument();
    });

    it("labels the AI allowance as the monthly one", () => {
      render(<Settings />);
      expect(
        screen.getByText("settings.subscription.perMonth"),
      ).toBeInTheDocument();
    });

    it("marks the active theme and language for assistive tech", () => {
      render(<Settings />);

      expect(screen.getByText("settings.light")).toHaveAttribute(
        "aria-pressed",
      );
      expect(screen.getByText("settings.english")).toHaveAttribute(
        "aria-pressed",
      );
    });
  });

  // The upgrade CTA is a hand-rolled button rather than the shared one, so it
  // never got the 44px phone floor the button scale applies: `py-2` around
  // 14px text measured 36px on a 390px screen.
  describe("upgrade call to action", () => {
    beforeEach(() => {
      mockFeatures.value.PAYMENTS = true;
      mockSubscription.value.nextPlan = "pro";
    });

    afterEach(() => {
      mockFeatures.value.PAYMENTS = false;
      mockSubscription.value.nextPlan = null;
    });

    it("gives it a 44px target on phones", () => {
      render(<Settings />);
      const upgrade = screen.getByText("settings.subscription.upgrade");

      expect(upgrade.className).toContain("max-sm:min-h-11");
    });
  });

  // Returning from checkout with ?subscription=success used to strip the
  // parameter and say nothing at all.
  describe("returning from checkout", () => {
    beforeEach(() => {
      mockNotify.showInfoNotification.mockClear();
      mockNotify.showSuccessNotification.mockClear();
      mockSearchParams.set.mockClear();
      mockInvalidateQueries.mockClear();
    });

    afterEach(() => {
      mockSearchParams.value = new URLSearchParams();
    });

    it("acknowledges the return without claiming the plan already changed", () => {
      mockSearchParams.value = new URLSearchParams("subscription=success");
      render(<Settings />);

      expect(mockNotify.showInfoNotification).toHaveBeenCalledWith(
        "settings.subscription.checkoutReceived",
      );
      expect(mockNotify.showSuccessNotification).not.toHaveBeenCalled();
    });

    it("refetches the subscription so the new plan shows up", () => {
      mockSearchParams.value = new URLSearchParams("subscription=success");
      render(<Settings />);

      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      });
    });

    it("clears the parameter out of the URL", () => {
      mockSearchParams.value = new URLSearchParams("subscription=success");
      render(<Settings />);

      expect(mockSearchParams.set).toHaveBeenCalledWith({});
    });

    it("says nothing on an ordinary visit", () => {
      render(<Settings />);
      expect(mockNotify.showInfoNotification).not.toHaveBeenCalled();
      expect(mockInvalidateQueries).not.toHaveBeenCalled();
    });
  });
});

describe("StageTemplates", () => {
  it("renders page title", () => {
    render(<StageTemplates />);
    expect(screen.getByText("stages.title")).toBeInTheDocument();
  });

  it("shows the recommended stages and empty pipeline placeholder when no data", () => {
    render(<StageTemplates />);
    // Recommended chips render (Huntr-style columns, no phases).
    expect(screen.getByText("stages.wishlist")).toBeInTheDocument();
    expect(screen.getByText("stages.offer")).toBeInTheDocument();
    // Empty pipeline placeholder instead of per-phase groups.
    expect(screen.getByText("stages.noStages")).toBeInTheDocument();
  });

  it("shows create button", () => {
    render(<StageTemplates />);
    expect(screen.getAllByText("stages.create").length).toBeGreaterThan(0);
  });
});
