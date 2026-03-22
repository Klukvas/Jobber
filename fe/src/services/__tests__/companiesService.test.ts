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

import { companiesService } from "../companiesService";

describe("companiesService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("list", () => {
    it("calls GET with all query params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await companiesService.list({
        limit: 10,
        offset: 5,
        sort_by: "name",
        sort_dir: "asc",
      });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("companies?");
      expect(url).toContain("limit=10");
      expect(url).toContain("offset=5");
      expect(url).toContain("sort_by=name");
      expect(url).toContain("sort_dir=asc");
    });

    it("omits undefined params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await companiesService.list({});

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toBe("companies?");
    });
  });

  describe("getById", () => {
    it("calls GET on companies/{id}", async () => {
      const mockCompany = { id: "c1", name: "Acme" };
      mockApiClient.get.mockResolvedValue(mockCompany);

      const result = await companiesService.getById("c1");

      expect(mockApiClient.get).toHaveBeenCalledWith("companies/c1");
      expect(result).toEqual(mockCompany);
    });
  });

  describe("create", () => {
    it("calls POST on companies", async () => {
      const input = { name: "New Corp" };
      mockApiClient.post.mockResolvedValue({ id: "c2", ...input });

      const result = await companiesService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith("companies", input);
      expect(result).toEqual({ id: "c2", name: "New Corp" });
    });
  });

  describe("update", () => {
    it("calls PATCH on companies/{id}", async () => {
      const data = { name: "Updated Corp" };
      mockApiClient.patch.mockResolvedValue({ id: "c1", ...data });

      const result = await companiesService.update("c1", data as never);

      expect(mockApiClient.patch).toHaveBeenCalledWith("companies/c1", data);
      expect(result).toEqual({ id: "c1", name: "Updated Corp" });
    });
  });

  describe("delete", () => {
    it("calls DELETE on companies/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await companiesService.delete("c1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("companies/c1");
    });
  });

  describe("getRelatedCounts", () => {
    it("calls GET on companies/{id}/related-counts", async () => {
      const mockCounts = { jobs_count: 3, applications_count: 7 };
      mockApiClient.get.mockResolvedValue(mockCounts);

      const result = await companiesService.getRelatedCounts("c1");

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "companies/c1/related-counts",
      );
      expect(result).toEqual(mockCounts);
    });
  });

  describe("toggleFavorite", () => {
    it("calls POST on companies/{id}/favorite", async () => {
      mockApiClient.post.mockResolvedValue({ is_favorite: true });

      const result = await companiesService.toggleFavorite("c1");

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "companies/c1/favorite",
      );
      expect(result).toEqual({ is_favorite: true });
    });
  });
});
