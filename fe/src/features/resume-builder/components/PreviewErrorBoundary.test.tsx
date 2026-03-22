import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { PreviewErrorBoundary } from "./PreviewErrorBoundary";

vi.mock("i18next", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@sentry/react", () => ({
  captureException: vi.fn(),
}));

function ThrowingChild(): never {
  throw new Error("Test render error");
}

function GoodChild() {
  return <div data-testid="good-child">All good</div>;
}

describe("PreviewErrorBoundary", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Suppress React error boundary console.error
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  it("renders children when there is no error", () => {
    render(
      <PreviewErrorBoundary>
        <GoodChild />
      </PreviewErrorBoundary>,
    );
    expect(screen.getByTestId("good-child")).toBeInTheDocument();
  });

  it("renders error fallback when child throws", () => {
    render(
      <PreviewErrorBoundary>
        <ThrowingChild />
      </PreviewErrorBoundary>,
    );
    expect(
      screen.getByText("resumeBuilder.preview.renderError"),
    ).toBeInTheDocument();
  });

  it("renders try again button in error state", () => {
    render(
      <PreviewErrorBoundary>
        <ThrowingChild />
      </PreviewErrorBoundary>,
    );
    expect(screen.getByText("common.tryAgain")).toBeInTheDocument();
  });
});
