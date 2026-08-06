import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { sharingService, buildShareUrl } from "../sharingService";

describe("sharingService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("create", () => {
    it("calls POST on shares", async () => {
      const mockShare = {
        id: "share-1",
        token: "abc",
        snapshot: {
          schema_version: 1,
          generated_at: "2026-08-06T12:00:00Z",
          overview: {
            total_applications: 10,
            active_applications: 4,
            closed_applications: 6,
            rejected_applications: 5,
            response_rate: 20,
            avg_days_to_first_response: 3,
          },
          funnel: [],
        },
        created_at: "2026-08-06T12:00:00Z",
      };
      mockApiClient.post.mockResolvedValue(mockShare);

      const result = await sharingService.create();

      expect(mockApiClient.post).toHaveBeenCalledWith("shares");
      expect(result).toEqual(mockShare);
    });
  });

  describe("list", () => {
    it("calls GET on shares", async () => {
      mockApiClient.get.mockResolvedValue([]);

      const result = await sharingService.list();

      expect(mockApiClient.get).toHaveBeenCalledWith("shares");
      expect(result).toEqual([]);
    });
  });

  describe("remove", () => {
    it("calls DELETE on shares/:id", async () => {
      mockApiClient.delete.mockResolvedValue({ message: "ok" });

      await sharingService.remove("share-1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("shares/share-1");
    });
  });

  describe("getPublic", () => {
    it("calls GET on public/shares/:token with encoding", async () => {
      const mockShare = { snapshot: null, created_at: "2026-08-06" };
      mockApiClient.get.mockResolvedValue(mockShare);

      const result = await sharingService.getPublic("to/ken");

      expect(mockApiClient.get).toHaveBeenCalledWith("public/shares/to%2Fken");
      expect(result).toEqual(mockShare);
    });
  });

  describe("buildShareUrl", () => {
    it("builds an absolute /s/:token URL", () => {
      expect(buildShareUrl("abc123")).toBe(
        `${window.location.origin}/s/abc123`,
      );
    });
  });
});
