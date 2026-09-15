import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";

const mockResetPassword = vi.hoisted(() => vi.fn());
let searchString = "?email=user@test.com&code=123456";

vi.mock("@/services/authService", () => ({
  authService: { resetPassword: mockResetPassword },
}));

vi.mock("react-router-dom", () => ({
  useSearchParams: () => [new URLSearchParams(searchString), vi.fn()],
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

import ResetPassword from "../ResetPassword";

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ResetPassword />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  searchString = "?email=user@test.com&code=123456";
});

describe("ResetPassword page", () => {
  it("shows the invalid-link state when no code is present", () => {
    searchString = "?email=user@test.com";
    renderPage();
    expect(screen.getByText("auth.invalidResetLink")).toBeInTheDocument();
    expect(screen.getByText("common.backToHome")).toBeInTheDocument();
  });

  // Both are required by `POST /auth/reset-password`, so a URL missing either
  // cannot reset anything — saying so beats spending one of the customer's
  // few attempts on a request the server will refuse.
  it("shows the invalid-link state when no email is present", () => {
    searchString = "?code=123456";
    renderPage();
    expect(screen.getByText("auth.invalidResetLink")).toBeInTheDocument();
  });

  it("shows the invalid-link state for a code that is not six characters", () => {
    searchString = "?email=user@test.com&code=1234";
    renderPage();
    expect(screen.getByText("auth.invalidResetLink")).toBeInTheDocument();
  });

  // A dead end with only "back to home" left the customer to find the reset
  // flow again themselves. This is the flow that issues a fresh code.
  it("offers a way to request a new code from the invalid-link state", () => {
    searchString = "";
    renderPage();
    expect(
      screen.getByText("auth.sendResetCode").closest("a"),
    ).toHaveAttribute("href", "/forgot-password");
  });

  it("renders the reset form when a code is present", () => {
    renderPage();
    expect(screen.getByText("auth.resetPasswordTitle")).toBeInTheDocument();
    expect(screen.getByLabelText("auth.newPassword")).toBeInTheDocument();
    expect(screen.getByLabelText("auth.confirmPassword")).toBeInTheDocument();
  });

  it("validates that a password is required", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );
    expect(screen.getByText("errors.required")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("rejects passwords shorter than 8 characters", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.type(screen.getByLabelText("auth.newPassword"), "short");
    await user.type(screen.getByLabelText("auth.confirmPassword"), "short");
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );
    expect(screen.getByText("errors.passwordTooShort")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  // The server counts runes. Four emoji are eight UTF-16 units but only four
  // code points, so a `.length < 8` check waved them through to a 422.
  it("rejects four astral characters, which are eight UTF-16 units", async () => {
    const user = userEvent.setup();
    renderPage();
    const fourEmoji = "😀😀😀😀";
    await user.click(screen.getByLabelText("auth.newPassword"));
    await user.paste(fourEmoji);
    await user.click(screen.getByLabelText("auth.confirmPassword"));
    await user.paste(fourEmoji);
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByText("errors.passwordTooShort")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("accepts an eight-code-point Unicode password", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue(undefined);
    renderPage();
    const eightCodePoints = "😀".repeat(8);
    await user.click(screen.getByLabelText("auth.newPassword"));
    await user.paste(eightCodePoints);
    await user.click(screen.getByLabelText("auth.confirmPassword"));
    await user.paste(eightCodePoints);
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(
      screen.queryByText("errors.passwordTooShort"),
    ).not.toBeInTheDocument();
    await waitFor(() => expect(mockResetPassword).toHaveBeenCalledTimes(1));
  });

  // bcrypt caps its input at 72 BYTES. A character count would let a 40-glyph
  // Cyrillic password (80 bytes) through to a server that then refuses it.
  it("rejects a password longer than 72 bytes before submitting", async () => {
    const user = userEvent.setup();
    renderPage();
    const tooLong = "a".repeat(73);
    await user.type(screen.getByLabelText("auth.newPassword"), tooLong);
    await user.type(screen.getByLabelText("auth.confirmPassword"), tooLong);
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByText("errors.passwordTooLong")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("counts multi-byte characters by byte, not by character", async () => {
    const user = userEvent.setup();
    renderPage();
    // 37 Cyrillic characters = 74 UTF-8 bytes, well under any 72-character cap.
    const multiByte = "я".repeat(37);
    await user.type(screen.getByLabelText("auth.newPassword"), multiByte);
    await user.type(screen.getByLabelText("auth.confirmPassword"), multiByte);
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByText("errors.passwordTooLong")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("accepts a password that is exactly 72 bytes", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue({});
    renderPage();
    const atLimit = "a".repeat(72);
    await user.type(screen.getByLabelText("auth.newPassword"), atLimit);
    await user.type(screen.getByLabelText("auth.confirmPassword"), atLimit);
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(
      screen.queryByText("errors.passwordTooLong"),
    ).not.toBeInTheDocument();
    expect(mockResetPassword).toHaveBeenCalled();
  });

  it("flags mismatched confirm password", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.type(screen.getByLabelText("auth.newPassword"), "longenough1");
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "different1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );
    expect(screen.getByText("errors.passwordsDontMatch")).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("submits with email, code and password then shows the success state", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue(undefined);
    renderPage();

    await user.type(screen.getByLabelText("auth.newPassword"), "validpass1");
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    await waitFor(() => expect(mockResetPassword).toHaveBeenCalledTimes(1));
    expect(mockResetPassword.mock.calls[0][0]).toEqual({
      email: "user@test.com",
      code: "123456",
      password: "validpass1",
    });
    expect(
      await screen.findByText("auth.passwordResetDone"),
    ).toBeInTheDocument();
  });

  it("maps INVALID_PASSWORD errors to the password field", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError("password must be at least 8 characters", "INVALID_PASSWORD", 422),
    );
    renderPage();

    await user.type(screen.getByLabelText("auth.newPassword"), "validpass1");
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    // The localised key, not the server's sentence — the backend answers in
    // English only, and this page is not always in English.
    expect(
      await screen.findByText("errors.passwordTooShort"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("password must be at least 8 characters"),
    ).not.toBeInTheDocument();
    // it is NOT shown in the top-level error banner
    expect(
      screen.queryByText("auth.passwordResetDone"),
    ).not.toBeInTheDocument();
  });

  it("shows a generic error banner for non-password errors", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError("code expired", "RESET_CODE_EXPIRED", 400),
    );
    renderPage();

    await user.type(screen.getByLabelText("auth.newPassword"), "validpass1");
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(await screen.findByText("code expired")).toBeInTheDocument();
  });

  it("maps PASSWORD_TOO_LONG to a localized password-field error", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError(
        "Password must be 72 bytes or fewer",
        "PASSWORD_TOO_LONG",
        400,
      ),
    );
    renderPage();

    await user.type(screen.getByLabelText("auth.newPassword"), "validpass1");
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    // Localized copy on the field, not the backend's English sentence in the
    // banner above the form.
    expect(
      await screen.findByText("errors.passwordTooLong"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Password must be 72 bytes or fewer"),
    ).not.toBeInTheDocument();
  });
});
