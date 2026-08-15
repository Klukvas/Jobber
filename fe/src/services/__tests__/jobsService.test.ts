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

import { jobsService } from "../jobsService";

describe("jobsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("list", () => {
    it("calls GET with query params", async () => {
      const mockData = { items: [], total: 0 };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await jobsService.list({
        limit: 10,
        offset: 5,
        status: "active",
        sort: "created_at:desc",
      });

      expect(mockApiClient.get).toHaveBeenCalledWith(
        expect.stringContaining("jobs?"),
      );
      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("limit=10");
      expect(url).toContain("offset=5");
      expect(url).toContain("status=active");
      expect(url).toContain("sort=created_at%3Adesc");
      expect(result).toEqual(mockData);
    });

    it("omits undefined params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await jobsService.list({});

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toBe("jobs?");
    });

    it("includes and encodes the search param when provided", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await jobsService.list({ search: "acme corp" });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("search=acme+corp");
    });

    it("omits an empty search param", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await jobsService.list({ search: "" });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).not.toContain("search=");
    });
  });

  describe("getById", () => {
    it("calls GET on jobs/{id}", async () => {
      const mockJob = { id: "j1", title: "Engineer" };
      mockApiClient.get.mockResolvedValue(mockJob);

      const result = await jobsService.getById("j1");

      expect(mockApiClient.get).toHaveBeenCalledWith("jobs/j1");
      expect(result).toEqual(mockJob);
    });
  });

  describe("create", () => {
    it("calls POST on jobs with data", async () => {
      const input = { title: "New Job" };
      const mockResponse = { id: "j2", ...input };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await jobsService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith("jobs", input);
      expect(result).toEqual(mockResponse);
    });
  });

  describe("update", () => {
    it("calls PATCH on jobs/{id}", async () => {
      const data = { title: "Updated" };
      const mockResponse = { id: "j1", title: "Updated" };
      mockApiClient.patch.mockResolvedValue(mockResponse);

      const result = await jobsService.update("j1", data as never);

      expect(mockApiClient.patch).toHaveBeenCalledWith("jobs/j1", data);
      expect(result).toEqual(mockResponse);
    });
  });

  describe("archive", () => {
    it("calls PATCH on jobs/{id} with is_archived: true", async () => {
      const mockResponse = { id: "j1", is_archived: true };
      mockApiClient.patch.mockResolvedValue(mockResponse);

      const result = await jobsService.archive("j1");

      expect(mockApiClient.patch).toHaveBeenCalledWith("jobs/j1", {
        is_archived: true,
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("unarchive", () => {
    it("calls PATCH on jobs/{id} with is_archived: false", async () => {
      const mockResponse = { id: "j1", is_archived: false };
      mockApiClient.patch.mockResolvedValue(mockResponse);

      const result = await jobsService.unarchive("j1");

      expect(mockApiClient.patch).toHaveBeenCalledWith("jobs/j1", {
        is_archived: false,
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("delete", () => {
    it("calls DELETE on jobs/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await jobsService.delete("j1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("jobs/j1");
    });
  });

  describe("move", () => {
    it("posts the target stage_template_id to jobs/{id}/move", async () => {
      const mockJob = {
        id: "j1",
        current_stage_template_id: "tpl-1",
        current_stage_name: "Screening",
      };
      mockApiClient.post.mockResolvedValue(mockJob);

      const result = await jobsService.move("j1", {
        stage_template_id: "tpl-1",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("jobs/j1/move", {
        stage_template_id: "tpl-1",
      });
      expect(result).toEqual(mockJob);
    });
  });

  describe("toggleFavorite", () => {
    it("calls POST on jobs/{id}/favorite", async () => {
      const mockResponse = { is_favorite: true };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await jobsService.toggleFavorite("j1");

      expect(mockApiClient.post).toHaveBeenCalledWith("jobs/j1/favorite");
      expect(result).toEqual(mockResponse);
    });
  });
});
