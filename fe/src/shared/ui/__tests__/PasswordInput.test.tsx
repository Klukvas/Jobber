import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { createRef } from "react";
import { PasswordInput } from "../PasswordInput";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

describe("PasswordInput", () => {
  it("renders as password type by default", () => {
    render(<PasswordInput data-testid="pw" />);
    expect(screen.getByTestId("pw")).toHaveAttribute("type", "password");
  });

  it("toggles to text when eye button is clicked", () => {
    render(<PasswordInput data-testid="pw" />);
    const toggle = screen.getByLabelText("auth.showPassword");
    fireEvent.click(toggle);
    expect(screen.getByTestId("pw")).toHaveAttribute("type", "text");
  });

  it("toggles back to password on second click", () => {
    render(<PasswordInput data-testid="pw" />);
    const toggle = screen.getByLabelText("auth.showPassword");
    fireEvent.click(toggle);
    const toggleHide = screen.getByLabelText("auth.hidePassword");
    fireEvent.click(toggleHide);
    expect(screen.getByTestId("pw")).toHaveAttribute("type", "password");
  });

  it("forwards ref", () => {
    const ref = createRef<HTMLInputElement>();
    render(<PasswordInput ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it("accepts placeholder", () => {
    render(<PasswordInput placeholder="Enter password" />);
    expect(screen.getByPlaceholderText("Enter password")).toBeInTheDocument();
  });
});
