import { useEffect, useState } from 'react';
import { useAuthStore } from '@/stores/authStore';

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [isInitialized, setIsInitialized] = useState(false);
  const { accessToken, refreshToken, user, clearAuth, setAuth } = useAuthStore();

  useEffect(() => {
    const initializeAuth = async () => {
      // If we have tokens but no access token, try to refresh
      if (!accessToken && refreshToken && user) {
        try {
          const response = await fetch(
            `${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/auth/refresh`,
            {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
              body: JSON.stringify({ refresh_token: refreshToken }),
            }
          );

          if (response.ok) {
            const data = await response.json();
            setAuth(data.access_token, data.refresh_token, user);
          } else {
            // Refresh failed, clear auth
            clearAuth();
          }
        } catch {
          // Refresh failed, clear auth
          clearAuth();
        }
      }

      setIsInitialized(true);
    };

    initializeAuth();
  }, []); // Run only once on mount

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
