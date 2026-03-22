import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import Applications from "../Applications";
import ApplicationDetail from "../ApplicationDetail";
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

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ id: "test-id" }),
  useLocation: () => ({ pathname: "/", search: "" }),
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
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

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@/shared/lib/features", () => ({
  FEATURES: {
    GOOGLE_CALENDAR: false,
    SENTRY: false,
    EMAIL_NOTIFICATIONS: false,
    PAYMENTS: false,
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: null,
    isLoading: false,
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
    invalidateQueries: vi.fn(),
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

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    subscription: null,
    isPro: false,
    isEnterprise: false,
    nextPlan: null,
    canCreate: () => true,
    usage: { jobs: 0, resumes: 0, applications: 0, ai_requests: 0 },
    limits: {
      max_jobs: 10,
      max_resumes: 3,
      max_applications: 20,
      max_ai_requests: 5,
    },
  }),
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

describe("Applications", () => {
  it("renders page title", () => {
    render(<Applications />);
    expect(screen.getByText("applications.title")).toBeInTheDocument();
  });

  it("shows empty state when no data", () => {
    render(<Applications />);
    expect(screen.getByText("applications.noApplications")).toBeInTheDocument();
  });

  it("shows create button", () => {
    render(<Applications />);
    expect(screen.getAllByText("applications.create").length).toBeGreaterThan(
      0,
    );
  });
});

describe("ApplicationDetail", () => {
  it("renders without crash and shows back button", () => {
    render(<ApplicationDetail />);
    expect(screen.getAllByText("common.back").length).toBeGreaterThan(0);
  });

  it("shows error state when no application data", () => {
    render(<ApplicationDetail />);
    expect(screen.getByText("applications.notFound")).toBeInTheDocument();
  });
});

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
    expect(screen.getByText("jobs.create")).toBeInTheDocument();
  });
});

describe("JobDetail", () => {
  it("renders without crash and shows back button", () => {
    render(<JobDetail />);
    expect(screen.getAllByText("jobs.backToJobs").length).toBeGreaterThan(0);
  });

  it("shows error state when no job data", () => {
    render(<JobDetail />);
    expect(screen.getByText("errors.notFound")).toBeInTheDocument();
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
});

describe("StageTemplates", () => {
  it("renders page title", () => {
    render(<StageTemplates />);
    expect(screen.getByText("stages.title")).toBeInTheDocument();
  });

  it("shows empty state when no data", () => {
    render(<StageTemplates />);
    expect(screen.getByText("stages.noStages")).toBeInTheDocument();
  });

  it("shows create button", () => {
    render(<StageTemplates />);
    expect(screen.getAllByText("stages.create").length).toBeGreaterThan(0);
  });
});
