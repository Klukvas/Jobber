import { changeSessionLifetime } from "@/shared/lib/sessionLifetime";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface User {
  id: string;
  email: string;
  name: string;
  locale: string;
  created_at: string;
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
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    },
  ),
);
