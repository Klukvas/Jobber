import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { subscriptionService } from "../subscriptionService";

describe("subscriptionService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getSubscription", () => {
    it("calls GET on subscription", async () => {
      const mockData = { plan: "free", limits: {} };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await subscriptionService.getSubscription();

      expect(mockApiClient.get).toHaveBeenCalledWith("subscription");
      expect(result).toEqual(mockData);
    });
  });

  describe("getCheckoutConfig", () => {
    it("calls GET on subscription/checkout-config", async () => {
      const mockConfig = { client_token: "tok_123" };
      mockApiClient.get.mockResolvedValue(mockConfig);

      const result = await subscriptionService.getCheckoutConfig();

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "subscription/checkout-config",
      );
      expect(result).toEqual(mockConfig);
    });
  });

  describe("createPortalSession", () => {
    it("calls POST on subscription/portal", async () => {
      const mockPortal = { url: "https://portal.example.com" };
      mockApiClient.post.mockResolvedValue(mockPortal);

      const result = await subscriptionService.createPortalSession();

      expect(mockApiClient.post).toHaveBeenCalledWith("subscription/portal");
      expect(result).toEqual(mockPortal);
    });
  });

  describe("changePlan", () => {
    it("calls POST on subscription/change-plan with plan", async () => {
      mockApiClient.post.mockResolvedValue(undefined);

      await subscriptionService.changePlan("pro");

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "subscription/change-plan",
        { plan: "pro" },
      );
    });
  });

  describe("cancelSubscription", () => {
    it("calls POST on subscription/cancel", async () => {
      mockApiClient.post.mockResolvedValue(undefined);

      await subscriptionService.cancelSubscription();

      expect(mockApiClient.post).toHaveBeenCalledWith("subscription/cancel");
    });
  });
});
