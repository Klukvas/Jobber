import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { Sidebar, APP_SIDEBAR_ID } from "../Sidebar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

/** Mutable so a test can put the phone slide-over in either state. */
const mockSidebarState = {
  isExpanded: true,
  isMobileOpen: false,
  toggleExpanded: vi.fn(),
  closeMobile: vi.fn(),
};

vi.mock("react-router-dom", () => ({
  NavLink: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to: string;
    className: unknown;
  }) => (
    <a
      href={to}
      className={
        typeof className === "function"
          ? className({ isActive: false })
          : className
      }
    >
      {children}
    </a>
  ),
  Link: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to: string;
    className?: string;
  }) => (
    <a href={to} className={className}>
      {children}
    </a>
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
  useSidebarStore: () => mockSidebarState,
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

  /**
   * On a phone this sidebar is the slide-over, and these rows are the whole of
   * the app's navigation. `px-3 py-2` around a 20px glyph measured 239x36 —
   * eight pixels short of the 44 WCAG 2.5.5 asks for, with 4px between one row
   * and the next. The minimum is released from `md` up, where the sidebar is
   * docked and driven by a pointer.
   */
  describe("tap targets while it is the phone slide-over", () => {
    it.each([
      "nav.applications",
      "nav.resumes",
      "nav.companies",
      "nav.coverLetters",
      "nav.stages",
      "nav.analytics",
    ])("gives the %s row a 44px minimum", (label) => {
      render(<Sidebar />);
      const row = screen.getByText(label).closest("a");

      expect(row?.className).toContain("min-h-11");
      expect(row?.className).toContain("md:min-h-0");
    });

    it("gives the account and logout rows the same minimum", () => {
      render(<Sidebar />);

      const account = screen.getByText("test@example.com").closest("a");
      const logout = screen.getByText("auth.logout").closest("button");

      expect(account?.className).toContain("min-h-11");
      expect(logout?.className).toContain("min-h-11");
    });

    // Mobile-only control, so the box costs the docked header nothing.
    it("gives the close button a 44x44 box", () => {
      render(<Sidebar />);
      const close = screen.getByLabelText("common.closeSidebar");

      expect(close.className).toContain("h-11");
      expect(close.className).toContain("w-11");
      expect(close.className).toContain("md:hidden");
    });

    // The collapse toggle is `hidden md:block` — it never appears on a phone,
    // so growing it would only have cost desktop density.
    it("leaves the desktop-only collapse toggle alone", () => {
      render(<Sidebar />);
      const collapse = screen.getByLabelText("common.collapseSidebar");

      expect(collapse.className).toContain("md:block");
      expect(collapse.className).toContain("p-2");
    });
  });
});

/**
 * Closed on a phone, this panel was only pushed off-screen by a transform —
 * which hides it from eyes and from nobody else. The first nine Tab stops on
 * every app page were its rows, sitting at x=-248 with the focus ring
 * invisible, and a screen reader read the whole thing out as page content.
 */
describe("Sidebar while it is closed on a phone", () => {
  afterEach(() => {
    mockSidebarState.isMobileOpen = false;
  });

  function aside(): HTMLElement {
    return document.querySelector("aside")!;
  }

  it("takes itself out of the tab order and the accessibility tree", () => {
    mockSidebarState.isMobileOpen = false;
    render(<Sidebar />);

    expect(aside().className).toMatch(/(^|\s)invisible(\s|$)/);
  });

  it("stays visible where it is docked", () => {
    mockSidebarState.isMobileOpen = false;
    render(<Sidebar />);

    expect(aside().className).toMatch(/md:visible/);
  });

  it("is reachable again once it is open", () => {
    mockSidebarState.isMobileOpen = true;
    render(<Sidebar />);

    expect(aside().className).not.toMatch(/(^|\s)invisible(\s|$)/);
  });

  // 32px of logo, and the control that leaves the app.
  it("gives the brand link a 44px height on phones", () => {
    render(<Sidebar />);
    const brand = screen.getByAltText("Jobber").closest("a");

    expect(brand?.className).toContain("min-h-11");
    expect(brand?.className).toContain("md:min-h-0");
  });
});

/**
 * The header's hamburger points `aria-controls` at this panel. The panel is
 * always rendered — closed on a phone it is only pushed off-screen — so the
 * reference has to resolve in both states.
 */
describe("Sidebar — the panel the header controls", () => {
  it.each([false, true])(
    "carries the controlled id whether the phone drawer is open (%s)",
    (open) => {
      mockSidebarState.isMobileOpen = open;
      const { container } = render(<Sidebar />);
      mockSidebarState.isMobileOpen = false;

      expect(container.querySelector(`#${APP_SIDEBAR_ID}`)).toBeInTheDocument();
    },
  );
});
