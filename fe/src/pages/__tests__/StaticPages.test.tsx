import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import Privacy from "../Privacy";
import Terms from "../Terms";
import Refund from "../Refund";
import NotFound from "../NotFound";

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

describe("Privacy", () => {
  it("renders without crash and shows heading", () => {
    render(<Privacy />);
    expect(screen.getByText("Privacy Policy")).toBeInTheDocument();
  });

  it("shows back to home link", () => {
    render(<Privacy />);
    expect(screen.getAllByText("common.backToHome").length).toBeGreaterThan(0);
  });
});

describe("Terms", () => {
  it("renders without crash and shows heading", () => {
    render(<Terms />);
    expect(
      screen.getByRole("heading", { level: 1, name: "Terms of Service" }),
    ).toBeInTheDocument();
  });

  it("shows back to home link", () => {
    render(<Terms />);
    expect(screen.getAllByText("common.backToHome").length).toBeGreaterThan(0);
  });
});

describe("Refund", () => {
  it("renders without crash and shows heading", () => {
    render(<Refund />);
    expect(screen.getByText("Refund Policy")).toBeInTheDocument();
  });

  it("shows back to home link", () => {
    render(<Refund />);
    expect(screen.getAllByText("common.backToHome").length).toBeGreaterThan(0);
  });
});

describe("NotFound", () => {
  it("renders without crash and shows 404", () => {
    render(<NotFound />);
    expect(screen.getByText("404")).toBeInTheDocument();
  });

  it("shows page title", () => {
    render(<NotFound />);
    expect(screen.getByText("notFound.title")).toBeInTheDocument();
  });

  it("shows description", () => {
    render(<NotFound />);
    expect(screen.getByText("notFound.description")).toBeInTheDocument();
  });

  it("shows navigation buttons", () => {
    render(<NotFound />);
    expect(screen.getByText("notFound.goToApp")).toBeInTheDocument();
  });
});
