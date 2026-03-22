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

import { applicationsService } from "../applicationsService";

describe("applicationsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("list", () => {
    it("calls GET with query params", async () => {
      const mockData = { items: [], total: 0 };
      mockApiClient.get.mockResolvedValue(mockData);

      const result = await applicationsService.list({
        limit: 20,
        offset: 10,
        sort_by: "last_activity",
        sort_dir: "desc",
      });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("applications?");
      expect(url).toContain("limit=20");
      expect(url).toContain("offset=10");
      expect(url).toContain("sort_by=last_activity");
      expect(url).toContain("sort_dir=desc");
      expect(result).toEqual(mockData);
    });

    it("omits undefined params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await applicationsService.list({});

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toBe("applications?");
    });
  });

  describe("getById", () => {
    it("calls GET on applications/{id}", async () => {
      const mockApp = { id: "a1" };
      mockApiClient.get.mockResolvedValue(mockApp);

      const result = await applicationsService.getById("a1");

      expect(mockApiClient.get).toHaveBeenCalledWith("applications/a1");
      expect(result).toEqual(mockApp);
    });
  });

  describe("create", () => {
    it("calls POST on applications", async () => {
      const input = { job_id: "j1" };
      const mockResponse = { id: "a1", job_id: "j1" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await applicationsService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith("applications", input);
      expect(result).toEqual(mockResponse);
    });
  });

  describe("update", () => {
    it("calls PATCH on applications/{id}", async () => {
      const data = { status: "interviewing" };
      mockApiClient.patch.mockResolvedValue({ id: "a1", ...data });

      const result = await applicationsService.update("a1", data as never);

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "applications/a1",
        data,
      );
      expect(result).toEqual({ id: "a1", ...data });
    });
  });

  describe("delete", () => {
    it("calls DELETE on applications/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await applicationsService.delete("a1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("applications/a1");
    });
  });

  describe("listStages", () => {
    it("calls GET on applications/{id}/stages", async () => {
      const mockStages = [{ id: "s1", name: "Applied" }];
      mockApiClient.get.mockResolvedValue(mockStages);

      const result = await applicationsService.listStages("a1");

      expect(mockApiClient.get).toHaveBeenCalledWith(
        "applications/a1/stages",
      );
      expect(result).toEqual(mockStages);
    });
  });

  describe("addStage", () => {
    it("calls POST on applications/{id}/stages", async () => {
      const input = { name: "Interview" };
      const mockResponse = { id: "s2", name: "Interview" };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await applicationsService.addStage("a1", input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "applications/a1/stages",
        input,
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe("completeStage", () => {
    it("calls PATCH on applications/{id}/stages/{stageId}/complete", async () => {
      const data = { notes: "Went well" };
      const mockResponse = { id: "s1", status: "completed" };
      mockApiClient.patch.mockResolvedValue(mockResponse);

      const result = await applicationsService.completeStage(
        "a1",
        "s1",
        data as never,
      );

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "applications/a1/stages/s1/complete",
        data,
      );
      expect(result).toEqual(mockResponse);
    });

    it("works without data argument", async () => {
      mockApiClient.patch.mockResolvedValue({ id: "s1" });

      await applicationsService.completeStage("a1", "s1");

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "applications/a1/stages/s1/complete",
        undefined,
      );
    });
  });

  describe("updateStage", () => {
    it("calls PATCH on applications/{id}/stages/{stageId}", async () => {
      const data = { status: "in_progress" };
      mockApiClient.patch.mockResolvedValue({ id: "s1", ...data });

      const result = await applicationsService.updateStage("a1", "s1", data);

      expect(mockApiClient.patch).toHaveBeenCalledWith(
        "applications/a1/stages/s1",
        data,
      );
      expect(result).toEqual({ id: "s1", ...data });
    });
  });

  describe("deleteStage", () => {
    it("calls DELETE on applications/{id}/stages/{stageId}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await applicationsService.deleteStage("a1", "s1");

      expect(mockApiClient.delete).toHaveBeenCalledWith(
        "applications/a1/stages/s1",
      );
    });
  });
});
