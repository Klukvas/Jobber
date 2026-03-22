import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  post: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { matchScoreService } from "../matchScoreService";

describe("matchScoreService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("checkMatch", () => {
    it("calls POST on match-score with job and resume IDs", async () => {
      const mockResponse = { score: 85, suggestions: [] };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await matchScoreService.checkMatch("j1", "r1");

      expect(mockApiClient.post).toHaveBeenCalledWith("match-score", {
        job_id: "j1",
        resume_id: "r1",
      });
      expect(result).toEqual(mockResponse);
    });

    it("propagates errors", async () => {
      mockApiClient.post.mockRejectedValue(new Error("Service unavailable"));

      await expect(
        matchScoreService.checkMatch("j1", "r1"),
      ).rejects.toThrow("Service unavailable");
    });
  });
});
