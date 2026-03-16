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
      // Read current auth state directly from the store so this effect
      // runs only once on mount without needing reactive dependencies.
      const { accessToken, refreshToken, user, setAuth, clearAuth } =
        useAuthStore.getState();

      // If we have tokens but no access token, try to refresh
      if (!accessToken && refreshToken && user) {
        try {
          const response = await fetch(
            `${import.meta.env.VITE_API_BASE_URL || "/api/v1"}/auth/refresh`,
            {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify({ refresh_token: refreshToken }),
            },
          );

          if (response.ok) {
            const data = await response.json();
            setAuth(data.access_token, data.refresh_token, user);
          } else {
            // Refresh failed, clear auth
            clearAuth();
          }
        } catch (err) {
          console.error("[AuthProvider] token refresh failed:", err);
          clearAuth();
        }
      }

      setIsInitialized(true);
    };

    initializeAuth();
  }, []); // Run only once on mount — values are read via getState() inside

  // Show loading state while checking auth
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
