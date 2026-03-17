import { useEffect, useState } from "react";
import { Outlet, Navigate, useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/authStore";
import { Sidebar } from "@/widgets/Sidebar";
import { Header } from "@/widgets/Header";
import { useOnboarding } from "@/features/onboarding/useOnboarding";
import { WelcomeWizard } from "@/features/onboarding/WelcomeWizard";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { SubscriptionSuccessModal } from "@/features/subscription/components/SubscriptionSuccessModal";
import { PRE_CHECKOUT_PLAN_KEY } from "@/features/subscription/usePaddleCheckout";
import type { SubscriptionPlan } from "@/shared/types/api";

const PLAN_RANK: Record<SubscriptionPlan, number> = {
  free: 0,
  pro: 1,
  enterprise: 2,
};

const POLL_INTERVAL_MS = 3_000;
const POLL_TIMEOUT_MS = 5 * 60_000;

export function AppLayout() {
  const { t } = useTranslation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { shouldShow, complete } = useOnboarding();
  const [, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { plan } = useSubscription();

  // Baseline plan stored before checkout. null = not in a post-checkout flow,
  // which prevents spurious modals on normal page loads.
  const [preCheckoutPlan, setPreCheckoutPlan] =
    useState<SubscriptionPlan | null>(null);
  const [upgradedPlan, setUpgradedPlan] = useState<SubscriptionPlan | null>(
    null,
  );
  const [isAwaitingUpgrade, setIsAwaitingUpgrade] = useState(false);

  // Detect upgrade: re-evaluates whenever plan, isAwaitingUpgrade, or
  // preCheckoutPlan change, so it works whether the plan was already "pro"
  // at mount time (fast webhook) or arrives later (slow webhook).
  useEffect(() => {
    console.log("[Checkout] detect effect:", {
      plan,
      isAwaitingUpgrade,
      preCheckoutPlan,
    });
    if (!isAwaitingUpgrade || preCheckoutPlan === null) return;
    if (PLAN_RANK[plan] > PLAN_RANK[preCheckoutPlan]) {
      setIsAwaitingUpgrade(false);
      setPreCheckoutPlan(null);
      setUpgradedPlan(plan);
    }
  }, [plan, isAwaitingUpgrade, preCheckoutPlan]); // eslint-disable-line react-hooks/exhaustive-deps

  // Polling: driven by isAwaitingUpgrade state so it's immune to React
  // StrictMode's double-invocation. State changes from the mount effect below
  // are only applied after the double-invocation completes, at which point
  // this effect re-runs cleanly with isAwaitingUpgrade = true.
  useEffect(() => {
    if (!isAwaitingUpgrade) return;

    queryClient.invalidateQueries({ queryKey: ["subscription"] });

    const intervalId = setInterval(() => {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
    }, POLL_INTERVAL_MS);

    const timeoutId = setTimeout(
      () => setIsAwaitingUpgrade(false),
      POLL_TIMEOUT_MS,
    );

    return () => {
      clearInterval(intervalId);
      clearTimeout(timeoutId);
    };
  }, [isAwaitingUpgrade, queryClient]); // eslint-disable-line react-hooks/exhaustive-deps

  // On mount: detect Paddle success redirect. Only sets state — no cleanup
  // needed, which makes this safe for StrictMode double-invocation (the second
  // run sees an empty URL and exits early, leaving state from the first run).
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    console.log("[Checkout] mount effect, search:", window.location.search);
    if (params.get("subscription") !== "success") return;

    setSearchParams({}, { replace: true });

    const stored = sessionStorage.getItem(
      PRE_CHECKOUT_PLAN_KEY,
    ) as SubscriptionPlan | null;
    sessionStorage.removeItem(PRE_CHECKOUT_PLAN_KEY);
    console.log(
      "[Checkout] stored baseline:",
      stored,
      "→ setting isAwaitingUpgrade",
    );

    setPreCheckoutPlan(stored ?? "free");
    setIsAwaitingUpgrade(true);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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

      <SubscriptionSuccessModal
        plan={upgradedPlan}
        onClose={() => setUpgradedPlan(null)}
      />

      {isAwaitingUpgrade && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <div className="flex flex-col items-center gap-4 rounded-xl border bg-card p-8 shadow-lg">
            <div className="h-10 w-10 animate-spin rounded-full border-4 border-muted border-t-blue-600" />
            <p className="text-sm font-medium text-muted-foreground">
              {t("settings.subscription.activating")}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
