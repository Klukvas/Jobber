import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Header } from "../Header";
import { APP_SIDEBAR_ID } from "../Sidebar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const toggleTheme = vi.fn();
const toggleMobile = vi.fn();

vi.mock("@/stores/themeStore", () => ({
  useThemeStore: () => ({
    theme: "dark",
    toggleTheme,
  }),
}));

// Mutable so a test can render the header with the drawer already open —
// the button's label, state and icon all depend on it.
const sidebarState = { toggleMobile, isMobileOpen: false };

vi.mock("@/stores/sidebarStore", () => ({
  useSidebarStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector(sidebarState),
}));

vi.mock("@/shared/ui/LanguageSwitcher", () => ({
  LanguageSwitcher: () => <div data-testid="lang-switcher" />,
}));

describe("Header", () => {
  it("renders mobile menu button", () => {
    render(<Header />);
    expect(screen.getByLabelText("common.openMenu")).toBeInTheDocument();
  });

  it("renders theme toggle button", () => {
    render(<Header />);
    expect(screen.getByLabelText("settings.switchToLight")).toBeInTheDocument();
  });

  it("renders language switcher", () => {
    render(<Header />);
    expect(screen.getByTestId("lang-switcher")).toBeInTheDocument();
  });

  it("calls toggleMobile when menu button is clicked", () => {
    render(<Header />);
    fireEvent.click(screen.getByLabelText("common.openMenu"));
    expect(toggleMobile).toHaveBeenCalled();
  });

  it("calls toggleTheme when theme button is clicked", () => {
    render(<Header />);
    fireEvent.click(screen.getByLabelText("settings.switchToLight"));
    expect(toggleTheme).toHaveBeenCalled();
  });

  /**
   * Both measured 40x40 on a phone. They are the shared Button's `icon` size,
   * so the fix lives there — this is the assertion that the header actually
   * gets it, since these two are the controls a thumb reaches for first.
   */
  it.each(["common.openMenu", "settings.switchToLight"])(
    "gives %s a 44x44 box on phones",
    (label) => {
      render(<Header />);
      const button = screen.getByLabelText(label);

      expect(button.className).toContain("max-sm:h-11");
      expect(button.className).toContain("max-sm:w-11");
    },
  );

  it("keeps both at 40x40 from sm up", () => {
    render(<Header />);

    expect(screen.getByLabelText("common.openMenu").className).toContain(
      "h-10 w-10",
    );
  });
});

/**
 * The hamburger is a disclosure. It carried a static "open menu" label in both
 * states and named nothing it controlled, so assistive tech had no way to tell
 * an open drawer from a shut one.
 */
describe("Header — mobile menu disclosure", () => {
  beforeEach(() => {
    sidebarState.isMobileOpen = false;
  });

  it("reports the drawer as collapsed to start with", () => {
    render(<Header />);

    expect(screen.getByLabelText("common.openMenu")).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("reports it as expanded, and says so in the label, once it is open", () => {
    sidebarState.isMobileOpen = true;
    render(<Header />);

    const button = screen.getByLabelText("common.close");
    expect(button).toHaveAttribute("aria-expanded", "true");
    expect(screen.queryByLabelText("common.openMenu")).not.toBeInTheDocument();
  });

  // The panel itself is the sidebar, which is always in the DOM — its own
  // test pins the id onto it, this one pins the reference.
  it.each([false, true])("names the sidebar it controls (open=%s)", (open) => {
    sidebarState.isMobileOpen = open;
    render(<Header />);

    const button = screen.getByLabelText(
      open ? "common.close" : "common.openMenu",
    );

    expect(button).toHaveAttribute("aria-controls", APP_SIDEBAR_ID);
  });
});
