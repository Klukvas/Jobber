import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import JobDetail from "../JobDetail";

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

const mockJob = {
  id: "test-id",
  title: "Backend Engineer",
  company_id: null,
  url: null,
  source: null,
  description: "",
  notes: "existing note",
  status: "saved",
  is_favorite: false,
  applied_at: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  job_comments: [],
  resume: null,
};

const mutateSpy = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    const base = {
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    };
    if (queryKey[0] === "job") return { ...base, data: mockJob };
    if (queryKey[0] === "job-stages") return { ...base, data: [] };
    if (queryKey[0] === "companies")
      return { ...base, data: { items: [], total: 0 } };
    if (queryKey[0] === "resumes")
      return { ...base, data: { items: [], total: 0 } };
    if (queryKey[0] === "resume-builder") return { ...base, data: [] };
    return { ...base, data: null };
  },
  useMutation: () => ({
    mutate: mutateSpy,
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
      isAuthenticated: true,
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

function getNotesSaveButton(): HTMLButtonElement {
  const textarea = screen.getByPlaceholderText("jobs.notesPlaceholder");
  const cardContent = textarea.parentElement as HTMLElement;
  return within(cardContent).getByRole("button", {
    name: /common\.save/,
  }) as HTMLButtonElement;
}

describe("JobDetail notes block", () => {
  beforeEach(() => {
    mutateSpy.mockClear();
  });

  it("renders a disabled save button inside the notes card when notes are unchanged", () => {
    render(<JobDetail />);
    expect(getNotesSaveButton().disabled).toBe(true);
  });

  it("enables the notes save button after editing notes and saves on click", () => {
    render(<JobDetail />);
    const textarea = screen.getByPlaceholderText("jobs.notesPlaceholder");
    fireEvent.change(textarea, { target: { value: "updated note" } });

    const saveButton = getNotesSaveButton();
    expect(saveButton.disabled).toBe(false);

    fireEvent.click(saveButton);
    expect(mutateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ notes: "updated note" }),
    );
  });
});
