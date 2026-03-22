import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Header } from "../Header";

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

vi.mock("@/stores/sidebarStore", () => ({
  useSidebarStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ toggleMobile }),
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
});
