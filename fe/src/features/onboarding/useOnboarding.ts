import { useSyncExternalStore, useCallback } from "react";
import { useAuthStore } from "@/stores/authStore";

const STORAGE_KEY = "jobber-onboarding-completed";

// "Not now" is a different answer from "don't show me this again": it holds
// for the current browser session only, so the tour comes back next visit
// instead of being silently lost forever.
const SESSION_DISMISS_KEY = "jobber-onboarding-dismissed";

// --- Completion store ---
let completionListeners: Array<() => void> = [];

function emitCompletionChange() {
  for (const listener of completionListeners) {
    listener();
  }
}

function subscribeCompletion(listener: () => void) {
  completionListeners = [...completionListeners, listener];
  return () => {
    completionListeners = completionListeners.filter((l) => l !== listener);
  };
}

function getCompletionSnapshot(): boolean {
  return (
    localStorage.getItem(STORAGE_KEY) === "true" ||
    sessionStorage.getItem(SESSION_DISMISS_KEY) === "true"
  );
}

// --- Highlight store ---
let highlightedPath: string | null = null;
let highlightListeners: Array<() => void> = [];

function emitHighlightChange() {
  for (const listener of highlightListeners) {
    listener();
  }
}

function subscribeHighlight(listener: () => void) {
  highlightListeners = [...highlightListeners, listener];
  return () => {
    highlightListeners = highlightListeners.filter((l) => l !== listener);
  };
}

function getHighlightSnapshot(): string | null {
  return highlightedPath;
}

export function setOnboardingHighlight(path: string | null) {
  highlightedPath = path;
  emitHighlightChange();
}

export function useOnboardingHighlight(): string | null {
  return useSyncExternalStore(subscribeHighlight, getHighlightSnapshot);
}

// --- Main hook ---
export function useOnboarding() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const completed = useSyncExternalStore(
    subscribeCompletion,
    getCompletionSnapshot,
  );

  const shouldShow = isAuthenticated && !completed;

  /** "I'm done with the tour" — never shown again on this device. */
  const complete = useCallback(() => {
    localStorage.setItem(STORAGE_KEY, "true");
    setOnboardingHighlight(null);
    emitCompletionChange();
  }, []);

  /**
   * "Not now" — hides the tour for this browser session only. Pressing Escape
   * or clicking the backdrop used to run `complete`, so a stray keystroke on
   * the first screen permanently retired the onboarding with no visible
   * decision and no obvious way back.
   */
  const dismissForSession = useCallback(() => {
    sessionStorage.setItem(SESSION_DISMISS_KEY, "true");
    setOnboardingHighlight(null);
    emitCompletionChange();
  }, []);

  const restart = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    sessionStorage.removeItem(SESSION_DISMISS_KEY);
    emitCompletionChange();
  }, []);

  return { shouldShow, complete, dismissForSession, restart };
}
