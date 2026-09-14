import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";

import { profileService } from "@/services/profileService";
import { useAuthStore } from "@/stores/authStore";
import type { UserDTO } from "@/shared/types/api";

/**
 * The key the profile is cached under, scoped to the account it describes.
 *
 * The bare `["profile"]` was an account-independent name for account-specific
 * data. Signing out and back in as somebody else on the same tab hit the
 * cached entry inside its stale window and rendered the *previous* person's
 * name — and, because the query writes what it reads back into the persisted
 * auth store, saved them into the new session. Signing out now empties the
 * whole cache (see `shared/lib/session`); this makes the entry unreadable by
 * another account even if something ever slipped past that.
 */
export function profileQueryKey(userId: string | undefined) {
  return ["profile", userId ?? "anonymous"] as const;
}

/**
 * The signed-in account, read from the server.
 *
 * The auth store is a *cache* of it — written at sign-in, persisted to
 * localStorage, and otherwise never refreshed. The account screen showed
 * whatever that copy said, which could be months out of date, and there was no
 * query behind it to refresh from. This is that query.
 *
 * The cached user is returned while the request is in flight and if it fails,
 * so a flaky network degrades to "slightly stale" rather than to a blank card.
 */
export function useProfile() {
  const setAuth = useAuthStore((state) => state.setAuth);
  const cachedUser = useAuthStore((state) => state.user);

  const query = useQuery({
    queryKey: profileQueryKey(cachedUser?.id),
    queryFn: ({ signal }) => profileService.get({ signal }),
    staleTime: 60_000,
    // Nobody to read a profile for. The server would answer 401 and the
    // client would force a sign-out that has already happened.
    enabled: !!cachedUser,
  });

  // The sidebar, the Sentry context and the account card all read the account
  // off the store, so the authoritative copy is written back to it — but only
  // on a successful read. A failed refresh must never blank a usable cached
  // account.
  const fetched = query.data;
  useEffect(() => {
    if (fetched) setAuth(fetched);
  }, [fetched, setAuth]);

  return {
    user: (fetched ?? cachedUser) as UserDTO | null,
    isLoading: query.isLoading,
    isError: query.isError,
  };
}
