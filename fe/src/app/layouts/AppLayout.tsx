import { useCallback, useEffect, useState } from "react";
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
import { SupportButton } from "@/features/support/SupportButton";
import {
  forgetPreCheckoutPlan,
  onCheckoutCompleted,
  readPreCheckoutPlan,
  PRE_CHECKOUT_PLAN_KEY,
} from "@/features/subscription/checkoutSignals";
import { resetConsent } from "@/shared/lib/consent";
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

  // Read checkout return state once on mount (pure read — no side effects).
  //
  // Checkout itself is a popup on this very page, so the normal path is the
  // same-page event below, not this. What this covers is a *reload* while the
  // popup was open: that destroys the popup along with the page, and the
  // baseline left in sessionStorage is the only surviving trace of a purchase
  // that may well have gone through. The ?subscription=success parameter is
  // still honoured for the same reason — it costs nothing and the storefront
  // may append it. Polling is harmless either way: the success modal only
  // appears once the backend actually reports a higher plan.
  const [initialRedirect] = useState(() => {
    const stored = sessionStorage.getItem(
      PRE_CHECKOUT_PLAN_KEY,
    ) as SubscriptionPlan | null;
    const params = new URLSearchParams(window.location.search);
    const hasSuccessParam = params.get("subscription") === "success";
    if (!stored && !hasSuccessParam) return null;
    return {
      baseline: (stored ?? "free") as SubscriptionPlan,
      hasSuccessParam,
    };
  });

  const [preCheckoutPlan, setPreCheckoutPlan] =
    useState<SubscriptionPlan | null>(initialRedirect?.baseline ?? null);
  const [upgradedPlan, setUpgradedPlan] = useState<SubscriptionPlan | null>(
    null,
  );
  const [isAwaitingUpgrade, setIsAwaitingUpgrade] = useState(!!initialRedirect);

  // Detect upgrade during render (avoids setState-in-effect).
  // Self-terminating: once triggered, isAwaitingUpgrade becomes false.
  if (
    isAwaitingUpgrade &&
    preCheckoutPlan !== null &&
    PLAN_RANK[plan] > PLAN_RANK[preCheckoutPlan]
  ) {
    setIsAwaitingUpgrade(false);
    setPreCheckoutPlan(null);
    setUpgradedPlan(plan);
  }

  // Stops waiting and forgets the baseline, so nothing re-triggers on the next
  // page load. Used both by the overlay's own dismiss button and by the bfcache
  // restore below — a buyer who backed out is done, and must not be held behind
  // a spinner until the poll times out.
  const dismissAwaitingUpgrade = useCallback(() => {
    setIsAwaitingUpgrade(false);
    setPreCheckoutPlan(null);
    forgetPreCheckoutPlan();
  }, []);

  // The checkout popup closes without navigating anywhere, so a completed
  // purchase produces no page load for the mount-time read above to notice.
  // This is that missing signal — and it is only a signal: it starts the same
  // poll a reload would, and the success modal still waits for the backend to
  // report the higher plan. The provider's own callback is never treated as
  // payment.
  useEffect(() => {
    return onCheckoutCompleted(() => {
      setPreCheckoutPlan(readPreCheckoutPlan());
      setUpgradedPlan(null);
      setIsAwaitingUpgrade(true);
      // The baseline has been lifted into state; leaving it on disk would make
      // the next page load poll all over again.
      forgetPreCheckoutPlan();
    });
  }, []);

  // Pressing Back — or leaving for the Account Management Portal — can restore
  // this page straight from the bfcache: no remount, no effects, so nothing
  // else clears the baseline and the *next* full load would poll for a purchase
  // that never happened.
  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      dismissAwaitingUpgrade();
    };

    window.addEventListener("pageshow", handlePageShow);
    return () => window.removeEventListener("pageshow", handlePageShow);
  }, [dismissAwaitingUpgrade]);

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
  }, [isAwaitingUpgrade, queryClient]);

  // Clean up URL params and sessionStorage after the checkout return
  useEffect(() => {
    if (!initialRedirect) return;
    if (initialRedirect.hasSuccessParam) {
      setSearchParams({}, { replace: true });
    }
    forgetPreCheckoutPlan();
  }, [initialRedirect, setSearchParams]);

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
            <button
              type="button"
              onClick={resetConsent}
              className="cursor-pointer transition-colors hover:text-foreground"
            >
              {t("cookieConsent.settings")}
            </button>
            <span>
              &copy; {new Date().getFullYear()} {t("home.footer.copyright")}
            </span>
          </div>
        </footer>
      </div>

      <SupportButton />
      <WelcomeWizard open={shouldShow} onComplete={complete} />

      <SubscriptionSuccessModal
        plan={upgradedPlan}
        onClose={() => setUpgradedPlan(null)}
      />

      {isAwaitingUpgrade && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="checkout-activating-label"
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
        >
          <div className="flex flex-col items-center gap-4 rounded-xl border bg-card p-8 shadow-lg">
            <div
              aria-hidden="true"
              className="h-10 w-10 animate-spin rounded-full border-4 border-muted border-t-primary"
            />
            <p
              id="checkout-activating-label"
              role="status"
              className="text-sm font-medium text-muted-foreground"
            >
              {t("settings.subscription.activating")}
            </p>
            {/* A buyer who closed the checkout without paying is otherwise
                trapped behind this until the poll times out. Focused on open:
                the overlay covers the app, so its only escape must be the first
                thing a keyboard or screen-reader user reaches. */}
            <button
              type="button"
              autoFocus
              onClick={dismissAwaitingUpgrade}
              className="rounded-md border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
            >
              {t("settings.subscription.activatingDismiss")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
