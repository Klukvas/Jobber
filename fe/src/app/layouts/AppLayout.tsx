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
  readFreshPreCheckoutPlan,
} from "@/features/subscription/checkoutSignals";
import { resetConsent } from "@/shared/lib/consent";
import { PoweredByFluxLab } from "@/shared/ui/PoweredByFluxLab";
import { TAP_TARGET_INLINE } from "@/shared/ui/tapTarget";
import type { SubscriptionPlan } from "@/shared/types/api";

const PLAN_RANK: Record<SubscriptionPlan, number> = {
  free: 0,
  pro: 1,
  enterprise: 2,
};

/** How often the backend is re-read while a checkout return is pending. */
export const POLL_INTERVAL_MS = 3_000;
const POLL_TIMEOUT_MS = 5 * 60_000;

/**
 * The hosted checkout's post-purchase redirect parameter.
 *
 * A breadcrumb, never evidence. It is appended by a dashboard setting we do
 * not control, it survives bookmarking and sharing, and anybody can type it —
 * so the only thing it is allowed to do here is get itself removed from the
 * address bar. What a checkout actually happened is decided by the baseline
 * this app wrote before redirecting to the checkout.
 */
const SUBSCRIPTION_PARAM = "subscription";

const FOOTER_LINK = `${TAP_TARGET_INLINE} transition-colors hover:text-foreground`;

export function AppLayout() {
  const { t } = useTranslation();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { shouldShow, complete, dismissForSession } = useOnboarding();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { plan } = useSubscription();

  // Read the checkout return state once, on mount.
  //
  // Not a pure read, despite living in a `useState` initialiser: an expired or
  // unusable baseline is *deleted* as it is read, which is the whole point —
  // leaving it would let the same stale entry re-arm this overlay on every
  // later navigation. The initialiser runs once per mount and the removal is
  // idempotent, so a double-invoked initialiser in StrictMode changes nothing.
  //
  // Checkout is a full-page redirect to the provider, so this is the normal
  // path: the buyer comes back (`?subscription=success`) to a fresh page load,
  // and the baseline left in sessionStorage is the only trace of the purchase.
  //
  // The baseline is the *whole* arming condition. `?subscription=success` used
  // to arm it too, standing in a "free" baseline when none was stored — which
  // meant a bookmarked or shared success URL, opened by somebody who was
  // already on pro, read as an upgrade from free and congratulated them on a
  // purchase that never happened. Only a baseline this app wrote, minutes ago,
  // before redirecting to checkout, correlates a page load to a checkout.
  const [initialBaseline] = useState<SubscriptionPlan | null>(() =>
    readFreshPreCheckoutPlan(),
  );

  const [preCheckoutPlan, setPreCheckoutPlan] =
    useState<SubscriptionPlan | null>(initialBaseline);
  const [upgradedPlan, setUpgradedPlan] = useState<SubscriptionPlan | null>(
    null,
  );
  const [isAwaitingUpgrade, setIsAwaitingUpgrade] = useState(
    initialBaseline !== null,
  );

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

  // Pressing Back — or leaving for the customer portal — can restore
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

  // The baseline has been lifted into state above; leaving the copy on disk
  // would make the next page load poll all over again.
  useEffect(() => {
    if (initialBaseline === null) return;
    forgetPreCheckoutPlan();
  }, [initialBaseline]);

  // Take the provider's parameter back out of the address bar, whether or not
  // it correlated to anything.
  //
  // Read from the router rather than from `window.location`: the two disagree
  // after any client-side navigation, and a `setSearchParams` driven by the
  // window's copy would rewrite the wrong query string. Only this one key is
  // removed — the old `setSearchParams({})` threw away every other parameter
  // the page was carrying, so returning from checkout onto, say, a filtered
  // list silently reset the filters.
  useEffect(() => {
    if (!searchParams.has(SUBSCRIPTION_PARAM)) return;
    const remaining = new URLSearchParams(searchParams);
    remaining.delete(SUBSCRIPTION_PARAM);
    setSearchParams(remaining, { replace: true });
  }, [searchParams, setSearchParams]);

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
            {/* The same links as the landing footer, and the same 12px-text
                tap area — shared so the two rows cannot drift apart again. */}
            <Link to="/#faq" className={FOOTER_LINK}>
              {t("home.nav.faq")}
            </Link>
            <Link to="/terms" className={FOOTER_LINK}>
              {t("home.footer.terms")}
            </Link>
            <Link to="/privacy" className={FOOTER_LINK}>
              {t("home.footer.privacy")}
            </Link>
            <Link to="/refund" className={FOOTER_LINK}>
              {t("home.footer.refund")}
            </Link>
            <button
              type="button"
              onClick={resetConsent}
              className={FOOTER_LINK}
            >
              {t("cookieConsent.settings")}
            </button>
            <span>
              &copy; {new Date().getFullYear()} {t("home.footer.copyright")}
            </span>
            <PoweredByFluxLab className="hover:text-foreground" />
          </div>
        </footer>
      </div>

      <SupportButton />
      <WelcomeWizard
        open={shouldShow}
        onComplete={complete}
        onDismissForSession={dismissForSession}
      />

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
