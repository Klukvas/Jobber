import { QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { router } from "./router";
import { I18nextProvider } from "react-i18next";
import i18n from "@/shared/lib/i18n";
import { useEffect } from "react";
import { useThemeStore } from "@/stores/themeStore";
import { AuthProvider } from "./providers/AuthProvider";
import { GlobalErrorBoundary } from "@/shared/ui/GlobalErrorBoundary";
import { CookieConsent } from "@/shared/ui/CookieConsent";
import { Toaster } from "sonner";
import { getQueryClient } from "@/shared/lib/queryClient";

export function Providers() {
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
            <Toaster position="top-right" richColors closeButton />
            <CookieConsent />
          </AuthProvider>
        </QueryClientProvider>
      </I18nextProvider>
    </GlobalErrorBoundary>
  );
}
