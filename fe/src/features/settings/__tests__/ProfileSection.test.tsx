import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

const mockGet = vi.hoisted(() => vi.fn());

const ACCOUNT = {
  id: "u1",
  email: "alex@example.com",
  name: "Alex Jobseeker",
  locale: "en",
  created_at: "2026-01-01T00:00:00Z",
};

const authState = vi.hoisted(() => ({
  user: null as Record<string, string> | null,
  setAuth: vi.fn(),
}));

/** Stands in for the GET /profile query. */
const profileQuery = vi.hoisted(() => ({
  data: undefined as unknown,
  isLoading: false,
  isError: false,
}));

vi.mock("@/services/profileService", () => ({
  profileService: { get: mockGet },
}));
vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: typeof authState) => unknown) =>
    selector(authState),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));
vi.mock("@tanstack/react-query", () => ({
  // Minimal stand-in: the query result is driven by `profileQuery`, and the
  // queryFn is still called so "does this hook actually fetch?" is testable.
  useQuery: ({
    queryFn,
    enabled,
  }: {
    queryFn: (ctx: { signal: AbortSignal }) => Promise<unknown>;
    enabled?: boolean;
  }) => {
    if (enabled !== false) {
      void queryFn({ signal: new AbortController().signal });
    }
    return profileQuery;
  },
}));

import { ProfileSection } from "../ProfileSection";

beforeEach(() => {
  vi.clearAllMocks();
  authState.user = { ...ACCOUNT };
  profileQuery.data = undefined;
  profileQuery.isLoading = false;
  profileQuery.isError = false;
  mockGet.mockResolvedValue(ACCOUNT);
});

/**
 * The card exists to be *true*. The auth store is written once at sign-in and
 * persisted to localStorage, so what it holds can be months old — a name
 * changed anywhere else showed here as whatever was current back then. The
 * server's copy is what is displayed, and nothing on this card writes one.
 */
describe("ProfileSection", () => {
  it("asks the server for the account rather than trusting the cached copy", () => {
    render(<ProfileSection />);

    expect(mockGet).toHaveBeenCalled();
  });

  it("shows the server's account once it arrives", async () => {
    authState.user = { ...ACCOUNT, name: "Stale Name", email: "old@x.test" };
    profileQuery.data = ACCOUNT;

    render(<ProfileSection />);

    await waitFor(() =>
      expect(screen.getByLabelText("settings.profile.name")).toHaveValue(
        "Alex Jobseeker",
      ),
    );
    expect(screen.getByLabelText("settings.profile.email")).toHaveValue(
      "alex@example.com",
    );
  });

  it("falls back to the cached account while the read is in flight", () => {
    authState.user = { ...ACCOUNT, name: "Cached Name" };
    profileQuery.isLoading = true;

    render(<ProfileSection />);

    expect(screen.getByLabelText("settings.profile.name")).toHaveValue(
      "Cached Name",
    );
  });

  it("keeps showing the cached account when the read fails, and says so", () => {
    authState.user = { ...ACCOUNT, name: "Cached Name" };
    profileQuery.isError = true;

    render(<ProfileSection />);

    expect(screen.getByLabelText("settings.profile.name")).toHaveValue(
      "Cached Name",
    );
    expect(
      screen.getByText("settings.profile.refreshFailed"),
    ).toBeInTheDocument();
  });

  it("says nothing about a refresh that worked", () => {
    profileQuery.data = ACCOUNT;

    render(<ProfileSection />);

    expect(
      screen.queryByText("settings.profile.refreshFailed"),
    ).not.toBeInTheDocument();
  });

  // Both fields are the server's to change: the email is a login credential
  // that needs a verification round-trip, and there is no write endpoint for
  // the name either. An editable-looking field would be an offer the API
  // cannot keep.
  it("offers no way to change either field", () => {
    profileQuery.data = ACCOUNT;

    render(<ProfileSection />);

    expect(screen.getByLabelText("settings.profile.name")).toHaveAttribute(
      "readonly",
    );
    expect(screen.getByLabelText("settings.profile.email")).toHaveAttribute(
      "readonly",
    );
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("renders empty fields rather than crashing with no account at all", () => {
    authState.user = null;

    render(<ProfileSection />);

    expect(screen.getByLabelText("settings.profile.name")).toHaveValue("");
    expect(screen.getByLabelText("settings.profile.email")).toHaveValue("");
  });
});
