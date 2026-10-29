import { AICreditConfirmation } from "@/features/subscription/components/AICreditConfirmation";
import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { router } from "./router";
import { I18nextProvider } from "react-i18next";
import i18n from "@/shared/lib/i18n";
import { useEffect, useSyncExternalStore } from "react";
import { useThemeStore } from "@/stores/themeStore";
import { AuthProvider } from "./providers/AuthProvider";
import { GlobalErrorBoundary } from "@/shared/ui/GlobalErrorBoundary";
import { CookieConsent } from "@/shared/ui/CookieConsent";
import { Toaster } from "sonner";
import { getQueryClient } from "@/shared/lib/queryClient";

export function Providers() {
  // Subscribe outside RouterProvider so print navigation also unmounts overlays
  // and releases consent's body padding before the renderer measures the page.
  const pathname = useSyncExternalStore(
    router.subscribe,
    () => router.state.location.pathname,
  );
  const originKey = useSyncExternalStore(
    router.subscribe,
    () => router.state.location.key,
  );
  const isResumePrint = pathname.replace(/\/$/, "") === "/print/resume";
  const theme = useThemeStore((state) => state.theme);

  useEffect(() => {
    const root = window.document.documentElement;
    root.classList.remove("light", "dark");
    root.classList.add(theme);
  }, [theme]);

  return (
    <GlobalErrorBoundary>
      <I18nextProvider i18n={i18n}>
        <QueryClientProvider client={getQueryClient()}>
          <AuthProvider>
            <RouterProvider router={router} />
            {!isResumePrint && (
              <>
                <Toaster position="top-right" richColors closeButton />
                <CookieConsent />
                <AICreditConfirmation originKey={originKey} />
              </>
            )}
          </AuthProvider>
        </QueryClientProvider>
      </I18nextProvider>
    </GlobalErrorBoundary>
  );
}
