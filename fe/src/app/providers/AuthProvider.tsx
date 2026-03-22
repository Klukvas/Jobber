import { useEffect, useState } from "react";
import * as Sentry from "@sentry/react";
import { useAuthStore } from "@/stores/authStore";
import { apiClient } from "@/services/api";
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

      // If the user was previously authenticated, verify the session cookie
      // is still valid. The 401 interceptor in apiClient handles refresh.
      if (user) {
        try {
          await apiClient.get("ping");
        } catch {
          // apiClient interceptor already tried refresh; if we're here it failed
          clearAuth();
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
