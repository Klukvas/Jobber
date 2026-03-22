import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { LoginModal } from "../LoginModal";
import { RegisterModal } from "../RegisterModal";
import { ForgotPasswordModal } from "../ForgotPasswordModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    isError: false,
  }),
}));

vi.mock("react-hook-form", () => ({
  useForm: () => ({
    register: () => ({}),
    handleSubmit: (fn: () => void) => (e: { preventDefault: () => void }) => {
      e.preventDefault();
      fn();
    },
    getValues: () => ({}),
    setError: vi.fn(),
    formState: { errors: {} },
  }),
}));

vi.mock("@hookform/resolvers/zod", () => ({
  zodResolver: () => vi.fn(),
}));

vi.mock("@/services/authService", () => ({
  authService: {
    login: vi.fn(),
    register: vi.fn(),
    forgotPassword: vi.fn(),
    resetPassword: vi.fn(),
    verifyEmail: vi.fn(),
  },
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ setAuth: vi.fn() }),
}));

vi.mock("@/features/auth/hooks/useResendCode", () => ({
  useResendCode: () => ({
    resend: vi.fn(),
    cooldown: 0,
    canResend: true,
  }),
}));

vi.mock("@/services/api", () => ({
  ApiError: class extends Error {
    code: string;
    constructor(message: string, code: string) {
      super(message);
      this.code = code;
    }
  },
}));

vi.mock("@/shared/lib/validation", () => ({
  loginSchema: {},
  registerSchema: {},
}));

// ---------- LoginModal ----------
describe("LoginModal", () => {
  it("renders when open", () => {
    render(
      <LoginModal
        open={true}
        onOpenChange={vi.fn()}
        onSwitchToRegister={vi.fn()}
      />,
    );
    // "auth.login" appears as both title and submit button
    expect(screen.getAllByText("auth.login").length).toBeGreaterThanOrEqual(1);
  });

  it("returns null when closed", () => {
    const { container } = render(
      <LoginModal
        open={false}
        onOpenChange={vi.fn()}
        onSwitchToRegister={vi.fn()}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(
      <LoginModal
        open={true}
        onOpenChange={vi.fn()}
        onSwitchToRegister={vi.fn()}
      />,
    );
    expect(screen.getByText("auth.email")).toBeInTheDocument();
    expect(screen.getByText("auth.password")).toBeInTheDocument();
  });

  it("renders switch to register link", () => {
    render(
      <LoginModal
        open={true}
        onOpenChange={vi.fn()}
        onSwitchToRegister={vi.fn()}
      />,
    );
    expect(screen.getByText("auth.dontHaveAccount")).toBeInTheDocument();
  });
});

// ---------- RegisterModal ----------
describe("RegisterModal", () => {
  it("renders when open", () => {
    render(
      <RegisterModal
        open={true}
        onOpenChange={vi.fn()}
        onSwitchToLogin={vi.fn()}
      />,
    );
    // The register modal renders "auth.register" as both heading and button
    expect(screen.getAllByText("auth.register").length).toBeGreaterThanOrEqual(
      1,
    );
  });

  it("returns null when closed", () => {
    const { container } = render(
      <RegisterModal
        open={false}
        onOpenChange={vi.fn()}
        onSwitchToLogin={vi.fn()}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(
      <RegisterModal
        open={true}
        onOpenChange={vi.fn()}
        onSwitchToLogin={vi.fn()}
      />,
    );
    expect(screen.getByText("auth.email")).toBeInTheDocument();
    expect(screen.getByText("auth.password")).toBeInTheDocument();
  });
});

// ---------- ForgotPasswordModal ----------
describe("ForgotPasswordModal", () => {
  it("renders when open", () => {
    render(
      <ForgotPasswordModal
        open={true}
        onOpenChange={vi.fn()}
        onBackToLogin={vi.fn()}
      />,
    );
    expect(screen.getByText("auth.forgotPasswordTitle")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <ForgotPasswordModal
        open={false}
        onOpenChange={vi.fn()}
        onBackToLogin={vi.fn()}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders email input for first step", () => {
    render(
      <ForgotPasswordModal
        open={true}
        onOpenChange={vi.fn()}
        onBackToLogin={vi.fn()}
      />,
    );
    expect(screen.getByText("auth.email")).toBeInTheDocument();
  });
});
