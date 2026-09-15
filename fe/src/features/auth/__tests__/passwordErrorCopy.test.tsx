import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ApiError } from "@/services/api";
import i18n from "@/shared/lib/i18n";
import en from "@/shared/locales/en.json";
import ru from "@/shared/locales/ru.json";
import uk from "@/shared/locales/uk.json";

const authApi = vi.hoisted(() => ({
  register: vi.fn(),
  resetPassword: vi.fn(),
  verifyEmail: vi.fn(),
  login: vi.fn(),
}));

vi.mock("@/services/authService", () => ({ authService: authApi }));

let searchString = "?email=user@test.com&code=123456";
vi.mock("react-router-dom", () => ({
  useSearchParams: () => [new URLSearchParams(searchString), vi.fn()],
  useNavigate: () => vi.fn(),
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ setAuth: vi.fn() }),
}));

vi.mock("@/features/auth/hooks/useResendCode", () => ({
  useResendCode: () => ({
    resend: vi.fn(),
    secondsLeft: 0,
    canResend: true,
    isPending: false,
  }),
}));

import ResetPassword from "@/pages/ResetPassword";
import { RegisterModal } from "@/features/auth/modals/RegisterModal";

/** The copy each locale actually ships for a password the server refused. */
const TOO_SHORT: Record<string, string> = {
  en: en.errors.passwordTooShort,
  ru: ru.errors.passwordTooShort,
  uk: uk.errors.passwordTooShort,
};

/**
 * What the Go backend answers with. It has one language, and it is not the
 * customer's: `error.message` was rendered straight into the form, so a
 * Russian- or Ukrainian-speaking customer was told "password must be at least
 * 8 characters" in English, in the middle of an otherwise translated page.
 */
const BACKEND_MESSAGE = "password must be at least 8 characters";

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  searchString = "?email=user@test.com&code=123456";
});

afterEach(async () => {
  await i18n.changeLanguage("en");
});

describe.each(["en", "ru", "uk"] as const)(
  "a password the server refused, in %s",
  (language) => {
    beforeEach(async () => {
      await i18n.changeLanguage(language);
    });

    it("is explained in the reset form's own language", async () => {
      const user = userEvent.setup();
      authApi.resetPassword.mockRejectedValue(
        new ApiError(BACKEND_MESSAGE, "INVALID_PASSWORD", 400),
      );
      renderWithQuery(<ResetPassword />);

      await user.type(
        screen.getByLabelText(i18n.t("auth.newPassword")),
        "validpass1",
      );
      await user.type(
        screen.getByLabelText(i18n.t("auth.confirmPassword")),
        "validpass1",
      );
      await user.click(
        screen.getByRole("button", { name: i18n.t("auth.resetPassword") }),
      );

      expect(await screen.findByText(TOO_SHORT[language])).toBeInTheDocument();
      expect(screen.queryByText(BACKEND_MESSAGE)).not.toBeInTheDocument();
    });

    it("is explained in the registration form's own language", async () => {
      const user = userEvent.setup();
      authApi.register.mockRejectedValue(
        new ApiError(BACKEND_MESSAGE, "INVALID_PASSWORD", 400),
      );
      renderWithQuery(
        <RegisterModal open onOpenChange={vi.fn()} onSwitchToLogin={vi.fn()} />,
      );

      await user.type(
        screen.getByLabelText(i18n.t("auth.email")),
        "user@test.com",
      );
      await user.type(
        screen.getByLabelText(i18n.t("auth.password")),
        "validpass1",
      );
      await user.type(
        screen.getByLabelText(i18n.t("auth.confirmPassword")),
        "validpass1",
      );
      await user.click(
        screen.getByRole("button", { name: i18n.t("auth.register") }),
      );

      await waitFor(() =>
        expect(authApi.register).toHaveBeenCalledOnce(),
      );
      expect(await screen.findByText(TOO_SHORT[language])).toBeInTheDocument();
      expect(screen.queryByText(BACKEND_MESSAGE)).not.toBeInTheDocument();
    });
  },
);
