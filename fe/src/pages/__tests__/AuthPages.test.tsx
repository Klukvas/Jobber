import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import VerifyEmail from "../VerifyEmail";
import ResetPassword from "../ResetPassword";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({}),
  useLocation: () => ({ pathname: "/", search: "" }),
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  Navigate: () => null,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

vi.mock("@/services/authService", () => ({
  authService: {
    verifyEmail: vi.fn(),
    resetPassword: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  ApiError: class ApiError extends Error {
    code: string;
    status: number;
    constructor(message: string, code: string, status: number) {
      super(message);
      this.code = code;
      this.status = status;
    }
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: null,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    isSuccess: false,
    error: null,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

describe("VerifyEmail", () => {
  it("renders without crash", () => {
    render(<VerifyEmail />);
    // With no code in search params, shows error state
    expect(screen.getByText("auth.verificationFailed")).toBeInTheDocument();
  });

  it("shows verification failed description", () => {
    render(<VerifyEmail />);
    expect(
      screen.getByText("auth.verificationFailedDescription"),
    ).toBeInTheDocument();
  });

  it("shows back to home link", () => {
    render(<VerifyEmail />);
    expect(screen.getByText("common.backToHome")).toBeInTheDocument();
  });
});

describe("ResetPassword", () => {
  it("renders without crash", () => {
    render(<ResetPassword />);
    // With no code in search params, shows invalid link state
    expect(screen.getByText("auth.invalidResetLink")).toBeInTheDocument();
  });

  it("shows back to home link when no code", () => {
    render(<ResetPassword />);
    expect(screen.getByText("common.backToHome")).toBeInTheDocument();
  });
});
