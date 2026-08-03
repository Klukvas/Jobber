import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { commentsService } from "../commentsService";

describe("commentsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("create", () => {
    it("calls POST on comments with data", async () => {
      const input = { job_id: "a1", body: "Great progress" };
      const mockResponse = { id: "cm1", ...input };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await commentsService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith("comments", input);
      expect(result).toEqual(mockResponse);
    });
  });

  describe("listByJob", () => {
    it("calls GET on jobs/{id}/comments", async () => {
      const mockComments = [{ id: "cm1", body: "Note" }];
      mockApiClient.get.mockResolvedValue(mockComments);

      const result = await commentsService.listByJob("a1");

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "jobs/a1/comments",
      );
      expect(result).toEqual(mockComments);
    });
  });

  describe("delete", () => {
    it("calls DELETE on comments/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await commentsService.delete("cm1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("comments/cm1");
    });
  });
});
