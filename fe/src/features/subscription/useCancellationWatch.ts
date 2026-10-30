import { useCallback, useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { useSubscription } from "@/shared/hooks/useSubscription";

/** How often the subscription is re-read while a cancellation is unconfirmed. */
export const CANCEL_POLL_INTERVAL_MS = 3_000;

/**
 * How long to keep asking. Bounded so a webhook that never arrives cannot leave
 * the page polling; well past the few seconds one normally takes.
 */
export const CANCEL_POLL_TIMEOUT_MS = 60_000;

/**
 * Carries a cancellation from the provider's answer to the row that shows it.
 *
 * `POST /subscription/cancel` only asks the provider. The row changes when the
 * provider's webhook lands, which is after that request has returned, so the
 * refetch right behind it reads the old state and, with the data fresh for a
 * minute, nothing else asks again.
 *
 * `isCancelPending` ends with the row, not with the polling: the provider has
 * the cancellation either way, so "renews on" stays wrong until a later read
 * proves otherwise. A free plan counts as well, because a cancellation that
 * ended the subscription outright leaves no `cancel_at` to wait for.
 */
export function useCancellationWatch() {
  const queryClient = useQueryClient();
  const { subscription, plan } = useSubscription();
  const [isRequested, setIsRequested] = useState(false);

  const isCancelPending =
    isRequested && !subscription?.cancel_at && plan !== "free";

  useEffect(() => {
    if (!isCancelPending) return;

    const intervalId = setInterval(() => {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
    }, CANCEL_POLL_INTERVAL_MS);

    const timeoutId = setTimeout(
      () => clearInterval(intervalId),
      CANCEL_POLL_TIMEOUT_MS,
    );

    return () => {
      clearInterval(intervalId);
      clearTimeout(timeoutId);
    };
  }, [isCancelPending, queryClient]);

  const watchCancellation = useCallback(() => setIsRequested(true), []);

  return { isCancelPending, watchCancellation };
}
