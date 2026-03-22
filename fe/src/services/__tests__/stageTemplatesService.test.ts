import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { stageTemplatesService } from "../stageTemplatesService";

describe("stageTemplatesService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("list", () => {
    it("calls GET with query params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await stageTemplatesService.list({ limit: 50, offset: 10 });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("stage-templates?");
      expect(url).toContain("limit=50");
      expect(url).toContain("offset=10");
    });

    it("omits undefined params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await stageTemplatesService.list({});

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toBe("stage-templates?");
    });
  });

  describe("create", () => {
    it("calls POST on stage-templates", async () => {
      const input = { name: "Phone Screen", order: 1 };
      mockApiClient.post.mockResolvedValue({ id: "st1", ...input });

      const result = await stageTemplatesService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "stage-templates",
        input,
      );
      expect(result).toEqual({ id: "st1", name: "Phone Screen", order: 1 });
    });
  });

  describe("update", () => {
    it("calls PATCH on stage-templates/{id}", async () => {
      const data = { name: "Updated Stage" };
      mockApiClient.patch.mockResolvedValue({ id: "st1", ...data });

      const result = await stageTemplatesService.update("st1", data as never);

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "stage-templates/st1",
        data,
      );
      expect(result).toEqual({ id: "st1", name: "Updated Stage" });
    });
  });

  describe("delete", () => {
    it("calls DELETE on stage-templates/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await stageTemplatesService.delete("st1");

      expect(mockApiClient.delete).toHaveBeenCalledWith(
        "stage-templates/st1",
      );
    });
  });
});
