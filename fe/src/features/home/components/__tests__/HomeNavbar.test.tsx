import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { HomeNavbar } from "../HomeNavbar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useLocation: () => ({ pathname: "/", search: "", hash: "" }),
  useNavigate: () => vi.fn(),
}));

vi.mock("@/stores/themeStore", () => ({
  useThemeStore: () => ({
    theme: "dark",
    toggleTheme: vi.fn(),
  }),
}));

vi.mock("@/shared/ui/LanguageSwitcher", () => ({
  LanguageSwitcher: () => <div data-testid="lang-switcher" />,
}));

describe("HomeNavbar", () => {
  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders brand name", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("Jobber")).toBeInTheDocument();
  });

  it("renders nav links", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("home.nav.features")).toBeInTheDocument();
    expect(screen.getByText("home.nav.howItWorks")).toBeInTheDocument();
    expect(screen.getByText("home.nav.pricing")).toBeInTheDocument();
    expect(screen.getByText("blog.title")).toBeInTheDocument();
  });

  it("renders login and register buttons when not authenticated", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("auth.login")).toBeInTheDocument();
    expect(screen.getByText("auth.register")).toBeInTheDocument();
  });

  it("renders go-to-platform button when authenticated", () => {
    render(
      <HomeNavbar
        {...defaultProps}
        isAuthenticated={true}
      />,
    );
    expect(screen.getByText("home.hero.ctaGoPlatform")).toBeInTheDocument();
    expect(screen.queryByText("auth.login")).not.toBeInTheDocument();
  });

  it("renders theme toggle button", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByLabelText("settings.switchToLight")).toBeInTheDocument();
  });

  it("renders language switcher", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByTestId("lang-switcher")).toBeInTheDocument();
  });

  it("opens features dropdown when clicked", () => {
    render(<HomeNavbar {...defaultProps} />);
    const btn = screen.getByText("home.nav.features");
    fireEvent.click(btn);
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("home.features.applications.title")).toBeInTheDocument();
    expect(screen.getByText("home.features.resumeBuilder.title")).toBeInTheDocument();
    expect(screen.getByText("home.features.coverLetters.title")).toBeInTheDocument();
  });

  it("renders mobile menu toggle", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByLabelText("common.openMenu")).toBeInTheDocument();
  });

  it("opens mobile menu when toggle is clicked", () => {
    render(<HomeNavbar {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("common.openMenu"));
    // Mobile menu should now show duplicate nav items
    const allHowItWorks = screen.getAllByText("home.nav.howItWorks");
    expect(allHowItWorks.length).toBeGreaterThan(1);
  });
});
