import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { type ReactNode } from "react";
import { GlobalErrorBoundary } from "../GlobalErrorBoundary";

vi.mock("@sentry/react", () => ({
  captureException: vi.fn(),
}));

vi.mock("@/shared/lib/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

function ThrowingChild(): ReactNode {
  throw new Error("Test error");
}

describe("GlobalErrorBoundary", () => {
  // Suppress React error boundary console.error output
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when no error", () => {
    render(
      <GlobalErrorBoundary>
        <div>OK content</div>
      </GlobalErrorBoundary>,
    );
    expect(screen.getByText("OK content")).toBeInTheDocument();
  });

  it("renders error UI when child throws", () => {
    render(
      <GlobalErrorBoundary>
        <ThrowingChild />
      </GlobalErrorBoundary>,
    );
    expect(screen.getByText("errors.somethingWentWrong")).toBeInTheDocument();
    expect(screen.getByText("errors.unexpectedError")).toBeInTheDocument();
  });

  it("renders reload and home buttons on error", () => {
    render(
      <GlobalErrorBoundary>
        <ThrowingChild />
      </GlobalErrorBoundary>,
    );
    expect(screen.getByText("errors.reloadPage")).toBeInTheDocument();
    expect(screen.getByText("common.backToHome")).toBeInTheDocument();
  });
});
