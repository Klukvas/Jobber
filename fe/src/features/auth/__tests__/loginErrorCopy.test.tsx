import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { ApiError } from "@/services/api";
import i18n from "@/shared/lib/i18n";
import en from "@/shared/locales/en.json";
import ru from "@/shared/locales/ru.json";
import uk from "@/shared/locales/uk.json";

const authApi = vi.hoisted(() => ({
  login: vi.fn(),
  resendVerification: vi.fn(),
  verifyEmail: vi.fn(),
}));

vi.mock("@/services/authService", () => ({ authService: authApi }));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ setAuth: vi.fn() }),
}));

vi.mock("@/features/auth/hooks/useResendCode", () => ({
  useResendCode: () => ({
    resend: vi.fn(),
    cooldown: 0,
    isDisabled: false,
    isSuccess: false,
    isLimitReached: false,
    resendError: "",
  }),
}));

import { LoginModal } from "@/features/auth/modals/LoginModal";

/** The copy each locale ships for credentials the server refused. */
const INVALID_CREDENTIALS: Record<string, string> = {
  en: en.auth.invalidCredentials,
  ru: ru.auth.invalidCredentials,
  uk: uk.auth.invalidCredentials,
};

/** And for anything this app has nothing specific to say about. */
const GENERIC: Record<string, string> = {
  en: en.errors.somethingWentWrong,
  ru: ru.errors.somethingWentWrong,
  uk: uk.errors.somethingWentWrong,
};

/**
 * What the Go backend answers with. It speaks one language, and it is not
 * necessarily the customer's: `error.message` went straight into the form, so
 * a Russian- or Ukrainian-speaking visitor who mistyped a password was told
 * "Invalid email or password" in English, in the middle of a translated page.
 * `auth.invalidCredentials` was already translated in all three locales and
 * was not used anywhere.
 */
const BACKEND_MESSAGE = "Invalid email or password";

function renderLogin() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <LoginModal open onOpenChange={vi.fn()} onSwitchToRegister={vi.fn()} />
    </QueryClientProvider>,
  );
}

async function submitLogin(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(i18n.t("auth.email")), "user@test.com");
  await user.type(screen.getByLabelText(i18n.t("auth.password")), "validpass1");
  await user.click(screen.getByRole("button", { name: i18n.t("auth.login") }));
}

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(async () => {
  await i18n.changeLanguage("en");
});

describe.each(["en", "ru", "uk"] as const)(
  "credentials the server refused, in %s",
  (language) => {
    beforeEach(async () => {
      await i18n.changeLanguage(language);
    });

    it("is explained in the login form's own language", async () => {
      const user = userEvent.setup();
      authApi.login.mockRejectedValue(
        new ApiError(BACKEND_MESSAGE, "INVALID_CREDENTIALS", 401),
      );
      renderLogin();

      await submitLogin(user);

      expect(
        await screen.findByText(INVALID_CREDENTIALS[language]),
      ).toBeInTheDocument();
      expect(screen.queryByText(BACKEND_MESSAGE)).not.toBeInTheDocument();
    });

    it("never shows the API's error code", async () => {
      const user = userEvent.setup();
      authApi.login.mockRejectedValue(
        new ApiError(BACKEND_MESSAGE, "INVALID_CREDENTIALS", 401),
      );
      renderLogin();

      await submitLogin(user);

      await screen.findByText(INVALID_CREDENTIALS[language]);
      expect(document.body.textContent).not.toContain("INVALID_CREDENTIALS");
    });

    it("falls back to one generic line for a failure it cannot name", async () => {
      const user = userEvent.setup();
      authApi.login.mockRejectedValue(
        new ApiError("database is on fire", "INTERNAL_ERROR", 500),
      );
      renderLogin();

      await submitLogin(user);

      expect(await screen.findByText(GENERIC[language])).toBeInTheDocument();
      expect(screen.queryByText("database is on fire")).not.toBeInTheDocument();
      expect(document.body.textContent).not.toContain("INTERNAL_ERROR");
    });

    it("falls back to the same line when the request never reached the API", async () => {
      const user = userEvent.setup();
      authApi.login.mockRejectedValue(
        new ApiError("Failed to fetch", "NETWORK_ERROR", 0),
      );
      renderLogin();

      await submitLogin(user);

      expect(await screen.findByText(GENERIC[language])).toBeInTheDocument();
      expect(screen.queryByText("Failed to fetch")).not.toBeInTheDocument();
    });
  },
);

describe("an unverified email", () => {
  it("still takes its own path rather than the generic message", async () => {
    const user = userEvent.setup();
    authApi.login.mockRejectedValue(
      new ApiError("verify your email", "EMAIL_NOT_VERIFIED", 403),
    );
    renderLogin();

    await submitLogin(user);

    expect(
      await screen.findByText(i18n.t("auth.emailNotVerified")),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(i18n.t("errors.somethingWentWrong")),
    ).not.toBeInTheDocument();
  });
});
