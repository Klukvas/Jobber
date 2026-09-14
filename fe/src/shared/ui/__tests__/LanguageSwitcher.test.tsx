import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LanguageSwitcher } from "../LanguageSwitcher";

const changeLanguage = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage },
  }),
}));

describe("LanguageSwitcher", () => {
  it("renders toggle button", () => {
    render(<LanguageSwitcher />);
    expect(screen.getByLabelText("common.changeLanguage")).toBeInTheDocument();
  });

  it("shows language menu when clicked", () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByLabelText("common.changeLanguage"));
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("English")).toBeInTheDocument();
  });

  it("calls changeLanguage when a language is selected", () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByLabelText("common.changeLanguage"));
    fireEvent.click(screen.getByText("English"));
    expect(changeLanguage).toHaveBeenCalledWith("en");
  });

  it("hides menu after selection", () => {
    render(<LanguageSwitcher />);
    fireEvent.click(screen.getByLabelText("common.changeLanguage"));
    fireEvent.click(screen.getByText("English"));
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("has aria-expanded attribute", () => {
    render(<LanguageSwitcher />);
    const btn = screen.getByLabelText("common.changeLanguage");
    expect(btn).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(btn);
    expect(btn).toHaveAttribute("aria-expanded", "true");
  });

  // A `role="menu"` promises the keyboard: it takes focus when it opens, the
  // arrows move inside it, and Escape hands focus back to the control that
  // opened it. Without the last one, dismissing the menu left focus on a
  // node React had just unmounted, which the browser drops onto `<body>`.
  describe("keyboard", () => {
    function openMenu() {
      render(<LanguageSwitcher />);
      const trigger = screen.getByLabelText("common.changeLanguage");
      fireEvent.click(trigger);
      return trigger;
    }

    it("focuses the current language when the menu opens", () => {
      openMenu();
      expect(document.activeElement).toBe(screen.getByText("English"));
    });

    it("moves between languages with the arrow keys", () => {
      openMenu();
      const menu = screen.getByRole("menu");

      fireEvent.keyDown(menu, { key: "ArrowDown" });
      expect(document.activeElement).toBe(screen.getByText("Українська"));

      fireEvent.keyDown(menu, { key: "ArrowUp" });
      expect(document.activeElement).toBe(screen.getByText("English"));

      // Wraps, so the last item is one key away from the first.
      fireEvent.keyDown(menu, { key: "ArrowUp" });
      expect(document.activeElement).toBe(screen.getByText("Русский"));
    });

    it("hands focus back to the trigger on Escape", () => {
      const trigger = openMenu();

      fireEvent.keyDown(document, { key: "Escape" });

      expect(screen.queryByRole("menu")).not.toBeInTheDocument();
      expect(document.activeElement).toBe(trigger);
    });

    it("hands focus back to the trigger after a selection", () => {
      const trigger = openMenu();

      fireEvent.click(screen.getByText("Українська"));

      expect(document.activeElement).toBe(trigger);
    });

    it("names the menu it controls in both states", () => {
      render(<LanguageSwitcher />);
      const trigger = screen.getByLabelText("common.changeLanguage");
      const controls = trigger.getAttribute("aria-controls");

      expect(controls).toBeTruthy();
      expect(document.getElementById(controls ?? "")).toBeInTheDocument();

      fireEvent.click(trigger);
      expect(document.getElementById(controls ?? "")).toBeVisible();
    });
  });

  // The small size is the landing navbar's, and that navbar is still finger-
  // driven at 768px: it measured 36x36 there. Asserted on classes because
  // jsdom has no layout and the classes are what decide the size.
  it("keeps the small trigger at 44x44 up to the lg breakpoint", () => {
    render(<LanguageSwitcher iconSize="sm" />);
    const btn = screen.getByLabelText("common.changeLanguage");

    expect(btn.className).toMatch(/(^|\s)h-11(\s|$)/);
    expect(btn.className).toMatch(/(^|\s)w-11(\s|$)/);
    expect(btn.className).not.toMatch(/sm:h-9/);
    expect(btn.className).toMatch(/lg:h-9/);
  });
});
