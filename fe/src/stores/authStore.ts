import { create } from 'zustand';
import { persist } from 'zustand/middleware';

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
    (set) => ({
      user: null,
      isAuthenticated: false,
      setAuth: (user) =>
        set({ user, isAuthenticated: true }),
      clearAuth: () => {
        set({ user: null, isAuthenticated: false });
        localStorage.removeItem('jobber-auth');
      },
    }),
    {
      name: 'jobber-auth',
    }
  )
);
