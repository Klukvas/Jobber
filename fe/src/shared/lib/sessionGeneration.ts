/**
 * Which sign-in the shared cookie jar belongs to, agreed on by every tab.
 *
 * Tabs share one cookie jar and one Web Lock queue, but each has its own
 * module state. The lock orders cookie writes; it cannot tell a queued request
 * that the session it was started for has ended since. Tab 1 logging out A
 * while tab 2 signs in B left tab 3's queued logout to erase B's cookies, and
 * tab 3's refresh-and-replay to run as B with A's request body.
 *
 * A generation is an opaque token, replaced on every logout and every
 * successful login. A sign-in's generation also names the account it signed
 * in, so a tab can tell "the same person signed in again" (keep going) from
 * "someone else, or nobody, owns the cookies now" (sign out). Auth work
 * records the generation it started in and checks it again before it touches
 * the network; a mismatch means the session it belongs to is over.
 *
 * `localStorage` is the shared copy, read synchronously at each check.
 * A `storage` event (other tabs only) and a `BroadcastChannel` tell idle tabs
 * to drop the old session at once instead of at their next request; a tab
 * coming back from the background or the back/forward cache checks again,
 * since it may have missed both. While storage throws (blocked in some
 * private modes, or out of quota) this tab keeps its own copy, which the
 * channel still updates, and tries storage again on its next change. Without
 * either, each tab keeps its own generation, which is the per-tab logout
 * epoch this replaced.
 */

export type SessionGeneration = string | null;

const STORAGE_KEY = "cursopus-session-generation";
const CHANNEL_NAME = "cursopus-session-generation";
// Separates the random part from the account id. Ids are UUIDs, so it cannot
// occur inside either part.
const ACCOUNT_SEPARATOR = ":";

const listeners = new Set<() => void>();
let readTabAccount: () => string | null = () => null;

// False after a failed write: a value this tab could not replace must not be
// read back as current, or its own next generation would look stale. The next
// successful write turns it back on.
let isStorageUsable = true;
// Last generation seen without storage: own changes and broadcast ones.
let fallback: SessionGeneration = readStored(null);
// The generation this tab's auth state and query cache belong to.
let tabGeneration: SessionGeneration = fallback;
const channel = openChannel();
listenForChanges();

function readStored(otherwise: SessionGeneration): SessionGeneration {
  if (!isStorageUsable) return otherwise;
  try {
    return globalThis.localStorage.getItem(STORAGE_KEY);
  } catch {
    return otherwise;
  }
}

/** The account a generation signed in, or null for a logout or an unknown one. */
export function accountOfGeneration(generation: SessionGeneration): string | null {
  if (!generation) return null;
  const at = generation.indexOf(ACCOUNT_SEPARATOR);
  return at < 0 ? null : generation.slice(at + 1) || null;
}

/** Tells this module which account the tab currently shows. */
export function setTabAccountSource(source: () => string | null): void {
  readTabAccount = source;
}

/** The account this tab currently shows, if signed in. */
export function tabAccount(): string | null {
  return readTabAccount();
}

/** The generation the shared cookie jar currently belongs to. */
export function sharedSessionGeneration(): SessionGeneration {
  return readStored(fallback);
}

/** The generation this tab last signed in or out under. */
export function tabSessionGeneration(): SessionGeneration {
  return tabGeneration;
}

export function isSessionGenerationCurrent(
  generation: SessionGeneration,
): boolean {
  return generation === sharedSessionGeneration();
}

/**
 * True while the cookies still belong to the account a request was sent for:
 * its own generation, or a later sign-in of that same account. A refresh and a
 * replay are safe then; the body goes to the account that wrote it.
 */
export function isSameAccountSession(
  generation: SessionGeneration,
  account: string | null | undefined,
): boolean {
  if (isSessionGenerationCurrent(generation)) return true;
  return !!account && accountOfGeneration(sharedSessionGeneration()) === account;
}

/** False while another tab has moved the session on and this tab has not caught up. */
export function isTabSessionCurrent(): boolean {
  return isSessionGenerationCurrent(tabGeneration);
}

/**
 * Starts a new generation for a logout (no account) or a sign-in made by this
 * tab (the account it signed in).
 *
 * A tab that has fallen behind drops its old session first, so starting the
 * next one never skips that cleanup.
 */
export function startSessionGeneration(account: string | null = null): SessionGeneration {
  syncTabSession();
  const next = account
    ? `${crypto.randomUUID()}${ACCOUNT_SEPARATOR}${account}`
    : crypto.randomUUID();
  try {
    globalThis.localStorage.setItem(STORAGE_KEY, next);
    isStorageUsable = true;
  } catch {
    // The channel and this tab's own copy carry it until a write succeeds.
    isStorageUsable = false;
  }
  fallback = next;
  tabGeneration = next;
  try {
    channel?.postMessage(next);
  } catch (error) {
    console.error(
      "[session] could not announce a session change:",
      error instanceof Error ? error.message : "unknown error",
    );
  }
  return next;
}

/**
 * Brings this tab onto the shared generation.
 *
 * When the same account this tab shows has signed in again elsewhere (after an
 * expired session, say), the tab keeps its state and only adopts the new
 * generation. Otherwise listeners run while the tab still counts as behind, so
 * whatever they clear cannot overwrite state the newer session has already
 * shared (the persisted auth entry).
 */
export function syncTabSession(): void {
  const shared = sharedSessionGeneration();
  if (shared === tabGeneration) return;
  const incoming = accountOfGeneration(shared);
  if (incoming && incoming === readTabAccount()) {
    tabGeneration = shared;
    return;
  }
  try {
    listeners.forEach((listener) => listener());
  } finally {
    tabGeneration = shared;
  }
}

/** Runs when another tab logs out or signs in another account, before this tab adopts it. */
export function onSessionReplaced(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function openChannel(): BroadcastChannel | null {
  if (typeof BroadcastChannel !== "function") return null;
  try {
    const opened = new BroadcastChannel(CHANNEL_NAME);
    opened.onmessage = (event: MessageEvent<unknown>) => {
      if (typeof event.data !== "string") return;
      fallback = event.data;
      syncTabSession();
    };
    return opened;
  } catch {
    return null;
  }
}

function listenForChanges(): void {
  if (typeof window === "undefined") return;
  window.addEventListener("storage", (event) => {
    // A null key is `localStorage.clear()`, which drops the generation too.
    if (event.key === STORAGE_KEY || event.key === null) syncTabSession();
  });
  // A frozen or cached tab can miss both events; catch up when it is shown.
  window.addEventListener("pageshow", () => syncTabSession());
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") syncTabSession();
  });
}
