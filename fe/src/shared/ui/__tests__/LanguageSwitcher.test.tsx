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
});
