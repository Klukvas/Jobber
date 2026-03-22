import { useEffect, useState } from "react";
import * as Sentry from "@sentry/react";
import { useAuthStore } from "@/stores/authStore";
import { FEATURES } from "@/shared/lib/features";

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [isInitialized, setIsInitialized] = useState(false);
  const user = useAuthStore((s) => s.user);

  // Sync Sentry user context with auth state
  useEffect(() => {
    if (!FEATURES.SENTRY) return;
    if (user) {
      Sentry.setUser({ id: user.id, email: user.email });
    } else {
      Sentry.setUser(null);
    }
  }, [user]);

  useEffect(() => {
    const initializeAuth = async () => {
      const { user, clearAuth } = useAuthStore.getState();

      // If we have a stored user, verify the session is still valid
      // by making a lightweight request. If cookies are expired, the
      // 401 interceptor in api.ts will attempt a refresh automatically.
      if (user) {
        try {
          const response = await fetch(
            `${import.meta.env.VITE_API_BASE_URL || "/api/v1"}/ping`,
            { credentials: "include" },
          );
          if (response.status === 401) {
            // Try refresh
            const refreshResponse = await fetch(
              `${import.meta.env.VITE_API_BASE_URL || "/api/v1"}/auth/refresh`,
              { method: "POST", credentials: "include" },
            );
            if (!refreshResponse.ok) {
              clearAuth();
            }
          }
        } catch {
          // Network error — keep user state, will retry on next API call
        }
      }

      setIsInitialized(true);
    };

    initializeAuth();
  }, []);

  if (!isInitialized) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
