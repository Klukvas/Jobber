import { changeSessionLifetime } from "@/shared/lib/sessionLifetime";
import { isTabSessionCurrent } from "@/shared/lib/sessionGeneration";
import { create } from "zustand";
import { createJSONStorage, persist, type StateStorage } from "zustand/middleware";

interface User {
  id: string;
  email: string;
  name: string;
  locale: string;
  created_at: string;
}

/**
 * `jobber-auth` is shared by every tab. A tab that another tab has signed out
 * or switched still clears its own state, but must not write that over the
 * entry the newer session saved.
 */
function currentSessionOnly(storage: Storage): StateStorage {
  return {
    getItem: (name) => storage.getItem(name),
    setItem: (name, value) => {
      if (isTabSessionCurrent()) storage.setItem(name, value);
    },
    removeItem: (name) => {
      if (isTabSessionCurrent()) storage.removeItem(name);
    },
  };
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (user: User) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      setAuth: (user) => {
        if (!get().isAuthenticated || get().user?.id !== user.id) changeSessionLifetime(true);
        set({ user, isAuthenticated: true });
      },
      clearAuth: () => {
        changeSessionLifetime(false);
        set({ user: null, isAuthenticated: false });
        // The persist middleware has just written `{user:null,...}`; a signed
        // out browser should hold no `jobber-auth` entry at all.
        try {
          useAuthStore.persist.clearStorage();
        } catch {
          // storage unavailable (private mode / blocked): nothing to remove
        }
      },
    }),
    {
      name: "jobber-auth",
      storage: createJSONStorage(() => currentSessionOnly(localStorage)),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
);
