import { apiClient } from "./api";
import type {
  SubscriptionDTO,
  CheckoutConfigDTO,
  PortalSessionDTO,
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
};
