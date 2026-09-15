import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";

import { useProfile, profileQueryKey } from "../useProfile";
import { endSession } from "@/shared/lib/session";
import { getQueryClient } from "@/shared/lib/queryClient";
import { useAuthStore } from "@/stores/authStore";
import type { UserDTO } from "@/shared/types/api";

const ALICE: UserDTO = {
  id: "user-a",
  email: "alice@example.com",
  name: "Alice Applicant",
  locale: "en",
  created_at: "2026-01-01T00:00:00Z",
};

const BORYS: UserDTO = {
  id: "user-b",
  email: "borys@example.com",
  name: "Borys Bootcamp",
  locale: "uk",
  created_at: "2026-02-01T00:00:00Z",
};

/** Whoever the access token currently belongs to, as far as the API knows. */
const server = vi.hoisted(() => ({ account: null as UserDTO | null }));

vi.mock("@/services/profileService", () => ({
  profileService: {
    get: vi.fn(async () => {
      if (!server.account) throw new Error("no session");
      return server.account;
    }),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

/** The one thing every screen does with the profile: show whose it is. */
function WhoAmI() {
  const { user } = useProfile();
  return <span data-testid="name">{user?.name ?? "nobody"}</span>;
}

function renderApp() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <WhoAmI />
    </QueryClientProvider>,
  );
}

/** What signing in does: the server names the account and the store caches it. */
function signIn(account: UserDTO) {
  server.account = account;
  useAuthStore.getState().setAuth(account);
}

beforeEach(() => {
  server.account = null;
  // Calls only — the fetch implementation lives in the module factory and has
  // to survive, and the call count is what the "read once" tests assert on.
  vi.clearAllMocks();
  getQueryClient().clear();
  useAuthStore.getState().clearAuth();
});

afterEach(() => {
  getQueryClient().clear();
  useAuthStore.getState().clearAuth();
});

/**
 * One browser tab, two people.
 *
 * The profile was cached under a bare `["profile"]` with a minute of staleness
 * and written back into the persisted auth store on every successful read.
 * Signing out cleared the store and left the cache alone, so the next sign-in
 * on the same tab rendered the previous person's name — and then saved it into
 * the new session, where the sidebar, the settings form and the Sentry context
 * all read it.
 */
describe("signing out of one account and into another", () => {
  it("never shows or stores the previous account's profile", async () => {
    signIn(ALICE);
    const { rerender } = renderApp();
    await screen.findByText("Alice Applicant");

    // Sign out through the path the app actually uses.
    act(() => endSession());
    expect(useAuthStore.getState().user).toBeNull();

    act(() => signIn(BORYS));
    rerender(
      <QueryClientProvider client={getQueryClient()}>
        <WhoAmI />
      </QueryClientProvider>,
    );

    // Not "eventually": Alice must not appear even for the frame before the
    // new read lands, which is exactly how long the stale cache used to win.
    expect(screen.getByTestId("name")).not.toHaveTextContent(
      "Alice Applicant",
    );
    await screen.findByText("Borys Bootcamp");

    expect(useAuthStore.getState().user).toMatchObject({ id: BORYS.id });
  });

  it("leaves nothing of the old session in the cache", async () => {
    signIn(ALICE);
    renderApp();
    await screen.findByText("Alice Applicant");
    expect(
      getQueryClient().getQueryData(profileQueryKey(ALICE.id)),
    ).toBeDefined();

    act(() => endSession());

    // Nothing the old session fetched is still readable. (Mounted observers
    // immediately re-register their own empty entries, so the check is for
    // surviving *data*, not for an empty cache object.)
    expect(
      getQueryClient()
        .getQueryCache()
        .getAll()
        .filter((query) => query.state.data !== undefined),
    ).toEqual([]);
  });

  /**
   * The profile query used to be paired with a save that wrote the response
   * into the cache and then invalidated the very key it had just seeded — an
   * active invalidation, so the newly-correct entry was thrown away and read
   * again over the network. The write is gone; this pins the read side down,
   * because the same round trip is the one a screen pays for on every mount.
   */
  it("reads the account once per mount, not twice", async () => {
    const { profileService } = await import("@/services/profileService");
    signIn(ALICE);

    renderApp();
    await screen.findByText("Alice Applicant");
    // Anything a redundant refetch scheduled has had its turn by now.
    await act(async () => {
      await Promise.resolve();
    });

    expect(profileService.get).toHaveBeenCalledTimes(1);
  });

  it("serves a second mount from the cache rather than reading again", async () => {
    const { profileService } = await import("@/services/profileService");
    signIn(ALICE);
    const first = renderApp();
    await screen.findByText("Alice Applicant");
    first.unmount();

    renderApp();
    await screen.findByText("Alice Applicant");

    expect(profileService.get).toHaveBeenCalledTimes(1);
  });

  // Belt and braces for the case the cache survives anyway — a second tab, a
  // sign-in that did not go through `endSession`. The entry is named after the
  // account it describes, so it simply is not B's to read.
  it("cannot serve one account's cached profile to another", async () => {
    getQueryClient().setQueryData(profileQueryKey(ALICE.id), ALICE);

    signIn(BORYS);
    renderApp();

    await waitFor(() =>
      expect(screen.getByTestId("name")).toHaveTextContent("Borys Bootcamp"),
    );
    expect(screen.getByTestId("name")).not.toHaveTextContent(
      "Alice Applicant",
    );
  });
});
