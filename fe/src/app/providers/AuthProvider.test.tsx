import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { AuthProvider } from "./AuthProvider";
import { ApiError } from "@/services/api";

const mocks = vi.hoisted(() => ({
  apiGet: vi.fn(),
  clearAuth: vi.fn(),
  state: { user: null as Record<string, unknown> | null },
}));

vi.mock("@sentry/react", () => ({
  setUser: vi.fn(),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({ user: mocks.state.user }),
    {
      getState: () => ({
        user: mocks.state.user,
        clearAuth: mocks.clearAuth,
      }),
    },
  ),
}));

vi.mock("@/services/api", () => {
  class MockApiError extends Error {
    code: string;
    status: number;

    constructor(message: string, code: string, status: number) {
      super(message);
      this.name = "ApiError";
      this.code = code;
      this.status = status;
    }
  }

  return {
    apiClient: { get: mocks.apiGet },
    ApiError: MockApiError,
  };
});

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { SENTRY: false },
}));

const TEST_USER = {
  id: "user-1",
  email: "test@example.com",
  name: "Test User",
  locale: "en",
  created_at: "2026-01-01T00:00:00Z",
};

async function renderAndInitialize() {
  render(
    <AuthProvider>
      <div data-testid="child">Child Content</div>
    </AuthProvider>,
  );

  await waitFor(() => {
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });
}

describe("AuthProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.state.user = null;
    mocks.apiGet.mockResolvedValue({});
  });

  it("renders loading state initially, then children", async () => {
    await renderAndInitialize();
  });

  it("shows loading spinner before initialization", () => {
    const { container } = render(
      <AuthProvider>
        <div>Child</div>
      </AuthProvider>,
    );
    expect(container.innerHTML).not.toBe("");
  });

  it("does not verify the session when no user is stored", async () => {
    await renderAndInitialize();

    expect(mocks.apiGet).not.toHaveBeenCalled();
    expect(mocks.clearAuth).not.toHaveBeenCalled();
  });

  it("verifies the session and keeps auth when the check succeeds", async () => {
    mocks.state.user = TEST_USER;

    await renderAndInitialize();

    expect(mocks.apiGet).toHaveBeenCalledWith("session");
    expect(mocks.clearAuth).not.toHaveBeenCalled();
  });

  it("clears auth when the session check is rejected with 401", async () => {
    mocks.state.user = TEST_USER;
    mocks.apiGet.mockRejectedValue(
      new ApiError("Unauthorized", "UNAUTHORIZED", 401),
    );

    await renderAndInitialize();

    expect(mocks.clearAuth).toHaveBeenCalledTimes(1);
  });

  it("clears auth when the session check is rejected with 403", async () => {
    mocks.state.user = TEST_USER;
    mocks.apiGet.mockRejectedValue(new ApiError("Forbidden", "FORBIDDEN", 403));

    await renderAndInitialize();

    expect(mocks.clearAuth).toHaveBeenCalledTimes(1);
  });

  it("keeps auth on network errors so a flaky connection does not log the user out", async () => {
    mocks.state.user = TEST_USER;
    mocks.apiGet.mockRejectedValue(new TypeError("Failed to fetch"));

    await renderAndInitialize();

    expect(mocks.clearAuth).not.toHaveBeenCalled();
  });

  it("keeps auth on server errors (5xx)", async () => {
    mocks.state.user = TEST_USER;
    mocks.apiGet.mockRejectedValue(
      new ApiError("Internal error", "INTERNAL_ERROR", 500),
    );

    await renderAndInitialize();

    expect(mocks.clearAuth).not.toHaveBeenCalled();
  });

  it("keeps auth when the session endpoint is missing (404)", async () => {
    mocks.state.user = TEST_USER;
    mocks.apiGet.mockRejectedValue(
      new ApiError("Not found", "NOT_FOUND", 404),
    );

    await renderAndInitialize();

    expect(mocks.clearAuth).not.toHaveBeenCalled();
  });
});
