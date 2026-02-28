import { useSyncExternalStore, useCallback } from 'react';
import { useAuthStore } from '@/stores/authStore';

const STORAGE_KEY = 'jobber-onboarding-completed';

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
  return localStorage.getItem(STORAGE_KEY) === 'true';
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
  const completed = useSyncExternalStore(subscribeCompletion, getCompletionSnapshot);

  const shouldShow = isAuthenticated && !completed;

  const complete = useCallback(() => {
    localStorage.setItem(STORAGE_KEY, 'true');
    setOnboardingHighlight(null);
    emitCompletionChange();
  }, []);

  const restart = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    emitCompletionChange();
  }, []);

  return { shouldShow, complete, restart };
}
