import { useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { subscriptionService } from "@/services/subscriptionService";
import { useCheckout } from "@/features/subscription/useCheckout";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { showSuccessNotification } from "@/shared/lib/notifications";
import type { SubscriptionPlan } from "@/shared/types/api";

interface PlanSelectionOptions {
  /** Called once the provider accepted a plan change, so the caller can close. */
  onPlanChanged?: () => void;
}

/**
 * Turns "the user picked this plan" into the one request that is correct for
 * them: a checkout, or a change to the subscription they already pay for.
 *
 * A subscriber must never go through checkout again. The subscription row holds
 * a single provider subscription ID, so a second purchase would overwrite it —
 * the original would keep billing at the provider with nothing pointing at it,
 * uncancellable from either side. The backend answers such a request with 409;
 * this is the UI half of the same rule, so it never gets that far.
 *
 * Returning to free is not a plan change but a cancellation, and lives in the
 * manage-subscription flow. Selecting it here does nothing.
 */
export function usePlanSelection({ onPlanChanged }: PlanSelectionOptions = {}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { plan } = useSubscription();
  const {
    openCheckout,
    isReady: isCheckoutReady,
    isPending: isCheckoutPending,
    error: checkoutError,
  } = useCheckout();

  const {
    mutate: changePlan,
    isPending: isChangingPlan,
    error: changePlanError,
  } = useMutation({
    mutationFn: (target: SubscriptionPlan) =>
      subscriptionService.changePlan(target),
    onSuccess: () => {
      // The provider's webhook is what actually moves the plan; refetching here
      // only shortens the wait for the row to catch up.
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
      showSuccessNotification(t("settings.subscription.manage.planChanged"));
      onPlanChanged?.();
    },
  });

  const isSubscriber = plan !== "free";

  const canSelect = useCallback(
    (target: SubscriptionPlan) => target !== plan && target !== "free",
    [plan],
  );

  const selectPlan = useCallback(
    (target: SubscriptionPlan) => {
      if (!canSelect(target)) return;
      if (isSubscriber) {
        changePlan(target);
        return;
      }
      void openCheckout(target);
    },
    [canSelect, changePlan, isSubscriber, openCheckout],
  );

  return {
    selectPlan,
    canSelect,
    isSubscriber,
    // A plan change goes straight to the backend, so it needs none of the
    // checkout config the purchase path waits for.
    isReady: isSubscriber || isCheckoutReady,
    isPending: isSubscriber ? isChangingPlan : isCheckoutPending,
    /**
     * True from the moment a purchase starts until the browser leaves for the
     * provider's hosted page (or the request fails). A subscriber's plan change
     * never redirects.
     */
    isCheckoutRedirecting: !isSubscriber && isCheckoutPending,
    /** i18n key for whichever request failed, or null while nothing has. */
    errorMessageKey: changePlanError
      ? "settings.subscription.manage.changePlanError"
      : checkoutError
        ? "settings.subscription.checkoutError"
        : null,
  };
}
