import { apiClient } from "./api";
import type {
  SubscriptionDTO,
  CheckoutConfigDTO,
  PortalSessionDTO,
  SubscriptionPlan,
} from "@/shared/types/api";

export const subscriptionService = {
  async getSubscription(): Promise<SubscriptionDTO> {
    return apiClient.get<SubscriptionDTO>("subscription");
  },

  async getCheckoutConfig(): Promise<CheckoutConfigDTO> {
    return apiClient.get<CheckoutConfigDTO>("subscription/checkout-config");
  },

  async createPortalSession(): Promise<PortalSessionDTO> {
    return apiClient.post<PortalSessionDTO>("subscription/portal");
  },

  async changePlan(plan: SubscriptionPlan): Promise<void> {
    return apiClient.post("subscription/change-plan", { plan });
  },

  async cancelSubscription(): Promise<void> {
    return apiClient.post("subscription/cancel");
  },
};
