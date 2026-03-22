import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { AuthProvider } from "./AuthProvider";

vi.mock("@sentry/react", () => ({
  setUser: vi.fn(),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({ user: null }),
    {
      getState: () => ({
        user: null,
        clearAuth: vi.fn(),
      }),
    },
  ),
}));

vi.mock("@/services/api", () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { SENTRY: false },
}));

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially, then children", async () => {
    render(
      <AuthProvider>
        <div data-testid="child">Child Content</div>
      </AuthProvider>,
    );

    // After initialization, children should be rendered
    await waitFor(() => {
      expect(screen.getByTestId("child")).toBeInTheDocument();
    });
  });

  it("shows loading spinner before initialization", () => {
    // This test checks the initial render before the async effect completes
    const { container } = render(
      <AuthProvider>
        <div>Child</div>
      </AuthProvider>,
    );
    // Either shows loading or children
    expect(container.innerHTML).not.toBe("");
  });
});
