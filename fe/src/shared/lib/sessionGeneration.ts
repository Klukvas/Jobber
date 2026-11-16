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
 * successful login. Auth work records the generation it started in and checks
 * it again before it touches the network; a mismatch means the session it
 * belongs to is over.
 *
 * `localStorage` is the shared copy, read synchronously at each check.
 * A `storage` event (other tabs only) and a `BroadcastChannel` tell idle tabs
 * to drop the old session at once instead of at their next request. Once
 * storage throws on a read or a write (blocked in some private modes, or out
 * of quota), this tab stops trusting it and keeps its own copy, which the
 * channel still updates. Without either, each tab keeps its own generation,
 * which is the per-tab logout epoch this replaced.
 */

export type SessionGeneration = string | null;

const STORAGE_KEY = "cursopus-session-generation";
const CHANNEL_NAME = "cursopus-session-generation";

const listeners = new Set<() => void>();

// Cleared when a write fails. A value this tab can no longer replace must not
// be read back as current, or its own next generation would look stale.
let isStorageUsable = true;
// Last generation seen without storage: own changes and broadcast ones.
let fallback: SessionGeneration = readStored(null);
// The generation this tab's auth state and query cache belong to.
let tabGeneration: SessionGeneration = fallback;
const channel = openChannel();
listenForStorage();

function readStored(otherwise: SessionGeneration): SessionGeneration {
  if (!isStorageUsable) return otherwise;
  try {
    return globalThis.localStorage.getItem(STORAGE_KEY);
  } catch {
    return otherwise;
  }
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

/** False while another tab has moved the session on and this tab has not caught up. */
export function isTabSessionCurrent(): boolean {
  return isSessionGenerationCurrent(tabGeneration);
}

/**
 * Starts a new generation for a logout or a sign-in made by this tab.
 *
 * A tab that has fallen behind drops its old session first, so starting the
 * next one never skips that cleanup.
 */
export function startSessionGeneration(): SessionGeneration {
  syncTabSession();
  const next = crypto.randomUUID();
  try {
    globalThis.localStorage.setItem(STORAGE_KEY, next);
  } catch {
    // The channel and this tab's own copy carry it from here on.
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
 * Brings this tab onto the shared generation. Listeners run while the tab
 * still counts as behind, so whatever they clear cannot overwrite state the
 * newer session has already shared (the persisted auth entry).
 */
export function syncTabSession(): void {
  const shared = sharedSessionGeneration();
  if (shared === tabGeneration) return;
  try {
    listeners.forEach((listener) => listener());
  } finally {
    tabGeneration = shared;
  }
}

/** Runs when another tab logs out or signs in, before this tab adopts it. */
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

function listenForStorage(): void {
  if (typeof window === "undefined") return;
  window.addEventListener("storage", (event) => {
    // A null key is `localStorage.clear()`, which drops the generation too.
    if (event.key === STORAGE_KEY || event.key === null) syncTabSession();
  });
}
