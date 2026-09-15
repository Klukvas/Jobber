import { useCallback, useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { subscriptionService } from "@/services/subscriptionService";
import { returnFocusWhenReady } from "@/features/subscription/checkoutOverlayFocus";
import {
  forgetPreCheckoutPlan,
  notifyCheckoutCompleted,
  rememberPreCheckoutPlan,
} from "@/features/subscription/checkoutSignals";
import { openPopupCheckout } from "@/features/subscription/fastspringSbl";
import { FEATURES } from "@/shared/lib/features";
import type { CheckoutSessionDTO, SubscriptionPlan } from "@/shared/types/api";

/**
 * What the customer is told when a checkout cannot start.
 *
 * One message for every failure, and deliberately free of detail: the caller
 * maps it to localised copy, and the underlying cause — a rejected session, a
 * malformed response, a blocked script — is not something a buyer can act on
 * and not something worth leaking into the UI.
 */
const CHECKOUT_FAILED_MESSAGE = "The checkout could not be completed";

/**
 * The session id from a create-session response, or null when the response
 * cannot supply one.
 *
 * The declared return type promises a `session_id`; the network does not. A
 * 204, a proxy that dropped the body, or a backend answering `{}` all arrive
 * here, and reading straight through them threw a bare `TypeError` from inside
 * the try — an internal detail on its way to the customer, and a failure mode
 * nothing had actually decided how to handle.
 */
function sessionIdOf(session: CheckoutSessionDTO | null | undefined) {
  const id = session?.session_id;
  return typeof id === "string" && id.length > 0 ? id : null;
}

/**
 * The control that has the keyboard right now, or null when none has.
 *
 * `document.activeElement` falls back to `<body>` rather than to null, and
 * `<body>` is not somewhere focus can usefully be handed back to.
 */
function focusedControl(): HTMLElement | null {
  const active = document.activeElement;
  if (!(active instanceof HTMLElement)) return null;
  return active === document.body ? null : active;
}

/**
 * Whether the page has the keyboard nowhere useful.
 *
 * The same test the overlay uses on its way out, and the reason a failed
 * checkout may move focus at all: if anything on the page holds it, the buyer
 * has moved on and that decision is the newer one.
 */
function isFocusStranded(): boolean {
  const active = document.activeElement;
  return !active || active === document.body;
}

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

  // Cancels a focus return still waiting for the CTA to come back out of its
  // busy state, so an unmount does not leave an observer on the page.
  const cancelFocusReturnRef = useRef<(() => void) | null>(null);
  useEffect(() => () => cancelFocusReturnRef.current?.(), []);

  /**
   * Puts the keyboard back on the control the buyer clicked.
   *
   * A checkout that never opened has no overlay to release, and releasing one
   * is the only thing that used to hand focus back. Meanwhile the click itself
   * had already taken focus away: the CTA goes into its busy state, and a
   * disabled button is blurred by the browser. So a failed checkout left the
   * page with focus on `<body>` — the buyer read an error message they could
   * not tab to, at the top of a document they had scrolled away from.
   *
   * Safe to call on any failure path, including the ones where the overlay has
   * already restored focus itself: the move is skipped unless the page is
   * still holding none, and it waits for the CTA to be re-enabled rather than
   * calling `focus()` on a disabled button, which does nothing.
   */
  const restoreOpenerFocus = useCallback(
    (returnFocusTo: HTMLElement | null) => {
      if (!returnFocusTo) return;
      cancelFocusReturnRef.current?.();
      cancelFocusReturnRef.current = returnFocusWhenReady(
        returnFocusTo,
        isFocusStranded,
      );
    },
    [],
  );

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
      // Read here, synchronously inside the click, and not later: the state
      // update below puts the button into its busy state, and a disabled
      // button is blurred — by the time the provider's popup exists there is
      // nothing left on the page to say where the buyer came from.
      //
      // `<body>` is not an answer. It is what `document.activeElement` reports
      // when nothing is focused, and naming it would have the overlay "restore"
      // focus to the top of the page — indistinguishable from the stranding
      // this exists to prevent, except that it also suppresses the fallback.
      const returnFocusTo = focusedControl();

      isStartingRef.current = true;
      setIsPopupOpen(true);
      setError(null);

      try {
        // Record the current plan before the popup opens. Nothing else can tell
        // an upgrade apart from a page that simply already had one.
        //
        // Inside the try on purpose: this reads a cache and writes storage, and
        // when it sat outside, a throwing sessionStorage — private mode, a full
        // quota — escaped `openCheckout` with `isStartingRef` still set and the
        // busy state still on. The button was dead for the rest of the page,
        // with nothing on screen to say why.
        const cached = queryClient.getQueryData<{ plan: SubscriptionPlan }>([
          "subscription",
        ]);
        rememberPreCheckoutPlan(cached?.plan ?? "free");

        const session = await createSession(plan);
        const sessionId = sessionIdOf(session);
        if (!sessionId) {
          throw new Error(CHECKOUT_FAILED_MESSAGE);
        }

        await openPopupCheckout({
          storefront: config.storefront,
          environment: config.environment,
          sessionId,
          returnFocusTo,
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
              setError(new Error(CHECKOUT_FAILED_MESSAGE));
              restoreOpenerFocus(returnFocusTo);
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
        restoreOpenerFocus(returnFocusTo);
      }
    },
    [config, createSession, finish, queryClient, restoreOpenerFocus],
  );

  return {
    openCheckout,
    isReady: (config?.plans?.length ?? 0) > 0 && !!config?.storefront,
    /** True from the click until the popup closes, so the UI stays busy. */
    isPending: isPopupOpen,
    error,
  };
}
