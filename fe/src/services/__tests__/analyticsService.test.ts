import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { analyticsService } from "../analyticsService";

describe("analyticsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getOverview", () => {
    it("calls GET on analytics/overview", async () => {
      const mockData = {
        total_applications: 50,
        active_applications: 20,
        closed_applications: 30,
        response_rate: 0.4,
        avg_days_to_first_response: 5,
      };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await analyticsService.getOverview();

      expect(mockApiClient.get).toHaveBeenCalledWith("analytics/overview");
      expect(result).toEqual(mockData);
    });
  });

  describe("getFunnel", () => {
    it("calls GET on analytics/funnel", async () => {
      const mockData = { stages: [] };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await analyticsService.getFunnel();

      expect(mockApiClient.get).toHaveBeenCalledWith("analytics/funnel");
      expect(result).toEqual(mockData);
    });
  });

  describe("getStageTime", () => {
    it("calls GET on analytics/stages", async () => {
      const mockData = { stages: [] };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await analyticsService.getStageTime();

      expect(mockApiClient.get).toHaveBeenCalledWith("analytics/stages");
      expect(result).toEqual(mockData);
    });
  });

  describe("getResumeEffectiveness", () => {
    it("calls GET on analytics/resumes", async () => {
      const mockData = { resumes: [] };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await analyticsService.getResumeEffectiveness();

      expect(mockApiClient.get).toHaveBeenCalledWith("analytics/resumes");
      expect(result).toEqual(mockData);
    });
  });

  describe("getSourceAnalytics", () => {
    it("calls GET on analytics/sources", async () => {
      const mockData = { sources: [] };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await analyticsService.getSourceAnalytics();

      expect(mockApiClient.get).toHaveBeenCalledWith("analytics/sources");
      expect(result).toEqual(mockData);
    });
  });
});
