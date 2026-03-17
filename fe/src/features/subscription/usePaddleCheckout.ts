import { useCallback, useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

export const PRE_CHECKOUT_PLAN_KEY = "paddle_pre_checkout_plan";
import { subscriptionService } from "@/services/subscriptionService";
import { useAuthStore } from "@/stores/authStore";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

// Paddle.js types (loaded via script tag)
interface PaddleInstance {
  Checkout: {
    open: (options: {
      items: Array<{ priceId: string; quantity: number }>;
      customer?: { email: string };
      customData?: Record<string, string>;
      settings?: {
        successUrl?: string;
        theme?: string;
      };
    }) => void;
  };
}

interface PaddleGlobal {
  Initialize: (options: { token: string }) => void;
  Environment: { set: (env: "sandbox" | "production") => void };
  Checkout: PaddleInstance["Checkout"];
}

declare global {
  interface Window {
    Paddle?: PaddleGlobal;
  }
}

export function usePaddleCheckout() {
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const initializedRef = useRef(false);

  const { data: config } = useQuery({
    queryKey: ["checkout-config"],
    queryFn: subscriptionService.getCheckoutConfig,
    staleTime: 300_000, // 5 minutes
    enabled: FEATURES.PAYMENTS,
  });

  // Load and initialize Paddle.js
  useEffect(() => {
    if (!FEATURES.PAYMENTS || !config?.client_token || initializedRef.current)
      return;

    const existingScript = document.querySelector('script[src*="paddle.com"]');

    const initPaddle = () => {
      if (window.Paddle && !initializedRef.current) {
        if (config.environment === "sandbox") {
          window.Paddle.Environment.set("sandbox");
        }
        window.Paddle.Initialize({ token: config.client_token });
        initializedRef.current = true;
      }
    };

    if (existingScript) {
      initPaddle();
      return;
    }

    const script = document.createElement("script");
    script.src = "https://cdn.paddle.com/paddle/v2/paddle.js";
    script.async = true;
    script.onload = initPaddle;
    document.head.appendChild(script);
  }, [config]);

  const openCheckout = useCallback(
    (plan: SubscriptionPlan = "pro") => {
      const priceId = config?.prices?.[plan];
      if (!window.Paddle || !priceId) {
        console.warn("[Paddle] openCheckout blocked", {
          paddleLoaded: !!window.Paddle,
          priceId,
        });
        return;
      }

      // Save current plan before redirect so AppLayout can detect upgrade on return
      const subData = queryClient.getQueryData<{ plan: SubscriptionPlan }>([
        "subscription",
      ]);
      const baseline = subData?.plan ?? "free";
      console.log("[Checkout] saving baseline to sessionStorage:", baseline);
      sessionStorage.setItem(PRE_CHECKOUT_PLAN_KEY, baseline);

      window.Paddle.Checkout.open({
        items: [{ priceId, quantity: 1 }],
        customer: user?.email ? { email: user.email } : undefined,
        customData: user?.id ? { user_id: user.id } : undefined,
        settings: {
          successUrl:
            window.location.origin + "/app/applications?subscription=success",
        },
      });
    },
    [config, user],
  );

  return {
    openCheckout,
    isReady:
      !!config?.client_token && Object.keys(config?.prices ?? {}).length > 0,
  };
}
