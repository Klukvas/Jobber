import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { subscriptionService } from "@/services/subscriptionService";
import {
  forgetPreCheckoutPlan,
  rememberPreCheckoutPlan,
} from "@/features/subscription/checkoutSignals";
import { safeBillingUrl } from "@/features/subscription/safeHttpsUrl";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

/**
 * What the customer is told when a checkout cannot start.
 *
 * One message for every failure, deliberately free of detail: a rejected
 * session or a malformed URL is not something a buyer can act on. It only
 * ends up in the thrown Error; the UI shows `settings.subscription.checkoutError`.
 */
const CHECKOUT_FAILED_MESSAGE = "The checkout could not be completed";

/**
 * Drives the billing provider's hosted checkout.
 *
 * The backend creates the checkout and returns its URL; the browser leaves for
 * it with a full-page redirect and the provider sends the buyer back to
 * `?subscription=success`. The frontend holds no provider credentials or
 * product identifiers — the only thing it chooses is a plan name.
 */
export function useCheckout() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<Error | null>(null);
  const [isRedirecting, setIsRedirecting] = useState(false);

  // A second click must not start a second checkout, and React state settles a
  // tick too late to stop one.
  const isStartingRef = useRef(false);

  // Pressing Back from the provider can restore this page from the bfcache with
  // the busy state still set; without this the button would stay dead.
  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      isStartingRef.current = false;
      setIsRedirecting(false);
    };
    window.addEventListener("pageshow", handlePageShow);
    return () => window.removeEventListener("pageshow", handlePageShow);
  }, []);

  const { data: config } = useQuery({
    queryKey: ["checkout-config"],
    queryFn: subscriptionService.getCheckoutConfig,
    staleTime: 300_000, // 5 minutes
    enabled: FEATURES.PAYMENTS,
  });

  const { mutateAsync: createSession } = useMutation({
    mutationFn: subscriptionService.createCheckoutSession,
  });

  const openCheckout = useCallback(
    async (plan: SubscriptionPlan = "pro") => {
      if (!config?.plans?.includes(plan) || isStartingRef.current) {
        return;
      }

      isStartingRef.current = true;
      setIsRedirecting(true);
      setError(null);

      try {
        // Recorded before leaving: it is how the layout detects the upgrade
        // after the redirect back. Inside the try so a throwing sessionStorage
        // cannot leave the busy state stuck.
        const cached = queryClient.getQueryData<{ plan: SubscriptionPlan }>([
          "subscription",
        ]);
        rememberPreCheckoutPlan(cached?.plan ?? "free");

        const checkoutUrl = safeBillingUrl(
          (await createSession(plan))?.checkout_url,
        );
        if (!checkoutUrl) {
          throw new Error(CHECKOUT_FAILED_MESSAGE);
        }

        // Stay busy: the page is navigating away.
        window.location.assign(checkoutUrl);
      } catch (cause) {
        isStartingRef.current = false;
        setIsRedirecting(false);
        forgetPreCheckoutPlan();
        setError(cause instanceof Error ? cause : new Error(String(cause)));
      }
    },
    [config, createSession, queryClient],
  );

  return {
    openCheckout,
    isReady: (config?.plans?.length ?? 0) > 0,
    /** True from the click until the page navigates away or is restored. */
    isPending: isRedirecting,
    error,
  };
}
