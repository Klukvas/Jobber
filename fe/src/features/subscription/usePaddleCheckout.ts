import { useCallback, useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { subscriptionService } from "@/services/subscriptionService";
import { useAuthStore } from "@/stores/authStore";
import { FEATURES } from "@/shared/lib/features";
import type { SubscriptionPlan } from "@/shared/types/api";

// Paddle.js types (loaded via script tag)
interface PaddleInstance {
  Checkout: {
    open: (options: {
      items: Array<{ priceId: string }>;
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
  Initialize: (options: { token: string; environment?: string }) => void;
  Checkout: PaddleInstance["Checkout"];
}

declare global {
  interface Window {
    Paddle?: PaddleGlobal;
  }
}

export function usePaddleCheckout() {
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);
  const initializedRef = useRef(false);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
        window.Paddle.Initialize({
          token: config.client_token,
          environment: config.environment === "sandbox" ? "sandbox" : undefined,
        });
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

  // Clean up polling on unmount
  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) clearInterval(pollIntervalRef.current);
      if (pollTimeoutRef.current) clearTimeout(pollTimeoutRef.current);
    };
  }, []);

  const openCheckout = useCallback(
    (plan: SubscriptionPlan = "pro") => {
      const priceId = config?.prices?.[plan];
      if (!window.Paddle || !priceId) {
        console.warn("[Paddle] openCheckout blocked", {
          paddleLoaded: !!window.Paddle,
          priceId,
          config,
        });
        return;
      }

      window.Paddle.Checkout.open({
        items: [{ priceId }],
        customer: user?.email ? { email: user.email } : undefined,
        customData: user?.id ? { user_id: user.id } : undefined,
        settings: {
          successUrl:
            window.location.origin + "/app/applications?subscription=success",
        },
      });

      // Clear any existing poll before starting a new one
      if (pollIntervalRef.current) clearInterval(pollIntervalRef.current);
      if (pollTimeoutRef.current) clearTimeout(pollTimeoutRef.current);

      // Poll for subscription changes after checkout opens
      pollIntervalRef.current = setInterval(() => {
        queryClient.invalidateQueries({ queryKey: ["subscription"] });
      }, 5000);

      // Stop polling after 10 minutes
      pollTimeoutRef.current = setTimeout(() => {
        if (pollIntervalRef.current) {
          clearInterval(pollIntervalRef.current);
          pollIntervalRef.current = null;
        }
      }, 600_000);
    },
    [config, user, queryClient],
  );

  return {
    openCheckout,
    isReady:
      !!config?.client_token && Object.keys(config?.prices ?? {}).length > 0,
  };
}
