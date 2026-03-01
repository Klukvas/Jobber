import { useEffect } from "react";
import { Outlet, Navigate, useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/authStore";
import { Sidebar } from "@/widgets/Sidebar";
import { Header } from "@/widgets/Header";
import { useOnboarding } from "@/features/onboarding/useOnboarding";
import { WelcomeWizard } from "@/features/onboarding/WelcomeWizard";

export function AppLayout() {
  const { t } = useTranslation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { shouldShow, complete } = useOnboarding();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();

  // Handle Paddle checkout success callback
  const subscriptionParam = searchParams.get("subscription");
  useEffect(() => {
    if (subscriptionParam === "success") {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
      setSearchParams({}, { replace: true });
    }
  }, [subscriptionParam, queryClient, setSearchParams]);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="flex-1 overflow-auto p-4 md:p-6">
          <Outlet />
        </main>
        <footer className="border-t px-4 py-3">
          <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
            <Link
              to="/terms"
              className="transition-colors hover:text-foreground"
            >
              {t("home.footer.terms")}
            </Link>
            <Link
              to="/privacy"
              className="transition-colors hover:text-foreground"
            >
              {t("home.footer.privacy")}
            </Link>
            <Link
              to="/refund"
              className="transition-colors hover:text-foreground"
            >
              {t("home.footer.refund")}
            </Link>
            <span>
              &copy; {new Date().getFullYear()} {t("home.footer.copyright")}
            </span>
          </div>
        </footer>
      </div>
      <WelcomeWizard open={shouldShow} onComplete={complete} />
    </div>
  );
}
