import { useCallback, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { subscriptionService } from "@/services/subscriptionService";
import {
  forgetPreCheckoutPlan,
  notifyCheckoutCompleted,
  rememberPreCheckoutPlan,
} from "@/features/subscription/checkoutSignals";
import { openPopupCheckout } from "@/features/subscription/fastspringSbl";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

/**
 * Drives the billing provider's popup checkout.
 *
 * The app never leaves the page: the backend creates a session server-side and
 * the browser hands that session's opaque id to the provider's popup. The
 * frontend holds no provider credentials, no product identifiers and no buyer
 * details — the only thing it chooses is a plan name.
 *
 * The provider's script is loaded lazily, here, at the moment a purchase
 * actually starts: a visitor who never opens the checkout never downloads it.
 */
export function useCheckout() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<Error | null>(null);
  const [isPopupOpen, setIsPopupOpen] = useState(false);

  // A second click must not start a second checkout, and React state settles a
  // tick too late to stop one. The ref closes that window; the state above is
  // only what the UI renders.
  const isStartingRef = useRef(false);

  const { data: config } = useQuery({
    queryKey: ["checkout-config"],
    queryFn: subscriptionService.getCheckoutConfig,
    staleTime: 300_000, // 5 minutes
    enabled: FEATURES.PAYMENTS,
  });

  const { mutateAsync: createSession } = useMutation({
    mutationFn: subscriptionService.createCheckoutSession,
  });

  const finish = useCallback(() => {
    isStartingRef.current = false;
    setIsPopupOpen(false);
  }, []);

  const openCheckout = useCallback(
    async (plan: SubscriptionPlan = "pro") => {
      if (!config?.plans?.includes(plan) || !config.storefront) {
        return;
      }
      if (isStartingRef.current) {
        return;
      }
      isStartingRef.current = true;
      setIsPopupOpen(true);
      setError(null);

      // Record the current plan before the popup opens. Nothing else can tell
      // an upgrade apart from a page that simply already had one.
      const cached = queryClient.getQueryData<{ plan: SubscriptionPlan }>([
        "subscription",
      ]);
      rememberPreCheckoutPlan(cached?.plan ?? "free");

      try {
        const session = await createSession(plan);
        await openPopupCheckout({
          storefront: config.storefront,
          environment: config.environment,
          sessionId: session.session_id,
          handlers: {
            onClose: ({ completed }) => {
              finish();
              if (!completed) {
                // Closed without buying: drop the baseline so no later page
                // load waits for an upgrade that is not coming.
                forgetPreCheckoutPlan();
                return;
              }
              // An order reference is a cue to start watching, never proof of
              // payment — the webhook decides, and the layout polls for it.
              notifyCheckoutCompleted();
            },
            onError: () => {
              finish();
              forgetPreCheckoutPlan();
              setError(new Error("The checkout could not be completed"));
            },
          },
        });
      } catch (cause) {
        // The popup never opened — a failed session, a blocked script, a
        // timeout. Clearing the baseline keeps the next load quiet, and the
        // module caches no failed load, so clicking again genuinely retries.
        finish();
        forgetPreCheckoutPlan();
        setError(cause instanceof Error ? cause : new Error(String(cause)));
      }
    },
    [config, createSession, finish, queryClient],
  );

  return {
    openCheckout,
    isReady: (config?.plans?.length ?? 0) > 0 && !!config?.storefront,
    /** True from the click until the popup closes, so the UI stays busy. */
    isPending: isPopupOpen,
    error,
  };
}
