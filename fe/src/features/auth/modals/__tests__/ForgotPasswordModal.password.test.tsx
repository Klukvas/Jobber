import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";

const mockForgotPassword = vi.hoisted(() => vi.fn());
const mockResetPassword = vi.hoisted(() => vi.fn());

vi.mock("@/services/authService", () => ({
  authService: {
    forgotPassword: mockForgotPassword,
    resetPassword: mockResetPassword,
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/features/auth/hooks/useResendCode", () => ({
  useResendCode: () => ({
    resend: vi.fn(),
    cooldown: 0,
    isDisabled: false,
    isPending: false,
    isSuccess: false,
    isLimitReached: false,
    resendError: null,
  }),
}));

import { ForgotPasswordModal } from "../ForgotPasswordModal";

function renderModal() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ForgotPasswordModal
        open={true}
        onOpenChange={vi.fn()}
        onBackToLogin={vi.fn()}
      />
    </QueryClientProvider>,
  );
}

// Walks the modal from the email step to the code step, where the new
// password is entered.
async function goToCodeStep(user: ReturnType<typeof userEvent.setup>) {
  mockForgotPassword.mockResolvedValue(undefined);
  renderModal();
  await user.type(screen.getByLabelText("auth.email"), "user@test.com");
  await user.click(screen.getByRole("button", { name: "auth.sendResetCode" }));
  await screen.findByLabelText("auth.verificationCode");
  await user.type(screen.getByLabelText("auth.verificationCode"), "123456");
}

async function typePassword(
  user: ReturnType<typeof userEvent.setup>,
  value: string,
) {
  await user.click(screen.getByLabelText("auth.newPassword"));
  await user.paste(value);
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ForgotPasswordModal password length", () => {
  it("rejects a seven-character password", async () => {
    const user = userEvent.setup();
    await goToCodeStep(user);
    await typePassword(user, "1234567");
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "errors.passwordTooShort",
    );
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  // The server counts runes: four emoji are eight UTF-16 units but four code
  // points, so a `.length < 8` check let them through to a 422.
  it("rejects four astral characters, which are eight UTF-16 units", async () => {
    const user = userEvent.setup();
    await goToCodeStep(user);
    await typePassword(user, "😀😀😀😀");
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "errors.passwordTooShort",
    );
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("accepts an eight-code-point Unicode password", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue(undefined);
    await goToCodeStep(user);
    await typePassword(user, "😀".repeat(8));
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    await waitFor(() => expect(mockResetPassword).toHaveBeenCalledTimes(1));
    expect(mockResetPassword.mock.calls[0][0]).toEqual({
      email: "user@test.com",
      code: "123456",
      password: "😀".repeat(8),
    });
  });

  // bcrypt caps its input at 72 BYTES. A character count would let a 37-glyph
  // Cyrillic password (74 bytes) through to a server that then refuses it —
  // and the modal used to report that refusal as an invalid code.
  it("rejects a Unicode password longer than 72 bytes before submitting", async () => {
    const user = userEvent.setup();
    await goToCodeStep(user);
    await typePassword(user, "я".repeat(37));
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "errors.passwordTooLong",
    );
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("accepts a password that is exactly 72 bytes", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue(undefined);
    await goToCodeStep(user);
    await typePassword(user, "a".repeat(72));
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    await waitFor(() => expect(mockResetPassword).toHaveBeenCalledTimes(1));
  });

  it("maps a backend PASSWORD_TOO_LONG to the password error, not the code error", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError(
        "Password must be 72 bytes or fewer",
        "PASSWORD_TOO_LONG",
        400,
      ),
    );
    await goToCodeStep(user);
    await typePassword(user, "validpass1");
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "errors.passwordTooLong",
      ),
    );
  });
});
