import { useAuthStore } from "@/stores/authStore";
import { getQueryClient } from "@/shared/lib/queryClient";
import { onSessionReplaced, setTabAccountSource } from "@/shared/lib/sessionGeneration";

/**
 * Ends the signed-in session in this tab: the one place that does.
 *
 * Clearing the auth store was never enough. Almost everything the app has
 * fetched is *this account's* data, and React Query kept all of it — so the
 * next person to sign in on the same tab was served the previous one's cached
 * rows until each key happened to go stale. The profile was the sharpest case:
 * `useProfile` caches for a minute and writes what it reads back into the
 * persisted auth store, so signing in as B rendered A's name and then saved A
 * into B's session.
 *
 * The whole cache goes, not a list of keys. A list is a thing to forget to add
 * to, and the failure mode of forgetting is showing one customer another
 * customer's data. The cost of over-clearing is a refetch of the handful of
 * account-independent keys (the checkout config, the blog), which is nothing.
 *
 * This deliberately does not navigate. Where to go next differs — the app
 * shell routes to the landing page, a refused token refresh reloads it — and
 * that decision belongs to the caller.
 */
export function endSession(): void {
  useAuthStore.getState().clearAuth();
  getQueryClient().clear();
}

// Another tab logged out or signed in a different account: everything this tab
// holds belongs to a session that is over. It is signed out rather than
// switched to the new account, which it may not even know yet (the other tab
// saves it later). The same account signing in again keeps this tab as it is.
setTabAccountSource(() => useAuthStore.getState().user?.id ?? null);
onSessionReplaced(endSession);
