import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Sidebar } from "../Sidebar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  NavLink: ({ children, to, className }: { children: React.ReactNode; to: string; className: unknown }) => (
    <a
      href={to}
      className={typeof className === "function" ? className({ isActive: false }) : className}
    >
      {children}
    </a>
  ),
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useNavigate: () => vi.fn(),
  useLocation: () => ({ pathname: "/app/applications" }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("@/stores/sidebarStore", () => ({
  useSidebarStore: () => ({
    isExpanded: true,
    isMobileOpen: false,
    toggleExpanded: vi.fn(),
    closeMobile: vi.fn(),
  }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      user: { email: "test@example.com" },
      clearAuth: vi.fn(),
    }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: "free" }),
}));

vi.mock("@/services/authService", () => ({
  authService: { logout: vi.fn() },
}));

vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboardingHighlight: () => null,
}));

describe("Sidebar", () => {
  it("renders brand name", () => {
    render(<Sidebar />);
    expect(screen.getByText("Jobber")).toBeInTheDocument();
  });

  it("renders all navigation items when expanded", () => {
    render(<Sidebar />);
    expect(screen.getByText("nav.applications")).toBeInTheDocument();
    expect(screen.getByText("nav.resumes")).toBeInTheDocument();
    expect(screen.getByText("nav.companies")).toBeInTheDocument();
    expect(screen.getByText("nav.jobs")).toBeInTheDocument();
    expect(screen.getByText("nav.coverLetters")).toBeInTheDocument();
    expect(screen.getByText("nav.stages")).toBeInTheDocument();
    expect(screen.getByText("nav.analytics")).toBeInTheDocument();
  });

  it("renders user email", () => {
    render(<Sidebar />);
    expect(screen.getByText("test@example.com")).toBeInTheDocument();
  });

  it("renders plan badge", () => {
    render(<Sidebar />);
    expect(
      screen.getByText("settings.subscription.freePlan"),
    ).toBeInTheDocument();
  });

  it("renders logout button", () => {
    render(<Sidebar />);
    expect(screen.getByText("auth.logout")).toBeInTheDocument();
  });

  it("renders collapse sidebar button", () => {
    render(<Sidebar />);
    expect(screen.getByLabelText("common.collapseSidebar")).toBeInTheDocument();
  });

  it("renders close mobile button", () => {
    render(<Sidebar />);
    expect(screen.getByLabelText("common.closeSidebar")).toBeInTheDocument();
  });
});
