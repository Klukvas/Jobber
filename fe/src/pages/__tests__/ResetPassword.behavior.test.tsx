import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";

const mockResetPassword = vi.hoisted(() => vi.fn());
let searchString = "?email=user@test.com&code=abc123";

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
  searchString = "?email=user@test.com&code=abc123";
});

describe("ResetPassword page", () => {
  it("shows the invalid-link state when no code is present", () => {
    searchString = "?email=user@test.com";
    renderPage();
    expect(screen.getByText("auth.invalidResetLink")).toBeInTheDocument();
    expect(screen.getByText("common.backToHome")).toBeInTheDocument();
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

  it("flags mismatched confirm password", async () => {
    const user = userEvent.setup();
    renderPage();
    await user.type(
      screen.getByLabelText("auth.newPassword"),
      "longenough1",
    );
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "different1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );
    expect(
      screen.getByText("errors.passwordsDontMatch"),
    ).toBeInTheDocument();
    expect(mockResetPassword).not.toHaveBeenCalled();
  });

  it("submits with email, code and password then shows the success state", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockResolvedValue(undefined);
    renderPage();

    await user.type(
      screen.getByLabelText("auth.newPassword"),
      "validpass1",
    );
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
      code: "abc123",
      password: "validpass1",
    });
    expect(
      await screen.findByText("auth.passwordResetDone"),
    ).toBeInTheDocument();
  });

  it("maps INVALID_PASSWORD errors to the password field", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError("weak password", "INVALID_PASSWORD", 422),
    );
    renderPage();

    await user.type(
      screen.getByLabelText("auth.newPassword"),
      "validpass1",
    );
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(await screen.findByText("weak password")).toBeInTheDocument();
    // it is NOT shown in the top-level error banner
    expect(screen.queryByText("auth.passwordResetDone")).not.toBeInTheDocument();
  });

  it("shows a generic error banner for non-password errors", async () => {
    const user = userEvent.setup();
    mockResetPassword.mockRejectedValue(
      new ApiError("code expired", "RESET_CODE_EXPIRED", 400),
    );
    renderPage();

    await user.type(
      screen.getByLabelText("auth.newPassword"),
      "validpass1",
    );
    await user.type(
      screen.getByLabelText("auth.confirmPassword"),
      "validpass1",
    );
    await user.click(
      screen.getByRole("button", { name: "auth.resetPassword" }),
    );

    expect(await screen.findByText("code expired")).toBeInTheDocument();
  });
});
