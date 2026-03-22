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

import { resumesService } from "../resumesService";

describe("resumesService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("list", () => {
    it("calls GET with all query params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await resumesService.list({
        limit: 10,
        offset: 0,
        sort_by: "created_at",
        sort_dir: "desc",
      });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("resumes?");
      expect(url).toContain("limit=10");
      expect(url).toContain("offset=0");
      expect(url).toContain("sort_by=created_at");
      expect(url).toContain("sort_dir=desc");
    });

    it("includes offset=0 when explicitly set", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await resumesService.list({ limit: 5, offset: 0 });

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toContain("offset=0");
    });

    it("omits undefined params", async () => {
      mockApiClient.get.mockResolvedValue({ items: [], total: 0 });

      await resumesService.list({});

      const url = mockApiClient.get.mock.calls[0][0] as string;
      expect(url).toBe("resumes?");
    });
  });

  describe("getById", () => {
    it("calls GET on resumes/{id}", async () => {
      const mockResume = { id: "r1", title: "My Resume" };
      mockApiClient.get.mockResolvedValue(mockResume);

      const result = await resumesService.getById("r1");

      expect(mockApiClient.get).toHaveBeenCalledWith("resumes/r1");
      expect(result).toEqual(mockResume);
    });
  });

  describe("create", () => {
    it("calls POST on resumes", async () => {
      const input = { title: "New Resume" };
      mockApiClient.post.mockResolvedValue({ id: "r2", ...input });

      const result = await resumesService.create(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith("resumes", input);
      expect(result).toEqual({ id: "r2", title: "New Resume" });
    });
  });

  describe("update", () => {
    it("calls PATCH on resumes/{id}", async () => {
      const data = { title: "Updated Resume" };
      mockApiClient.patch.mockResolvedValue({ id: "r1", ...data });

      const result = await resumesService.update("r1", data as never);

      expect(mockApiClient.patch).toHaveBeenCalledWith("resumes/r1", data);
      expect(result).toEqual({ id: "r1", title: "Updated Resume" });
    });
  });

  describe("delete", () => {
    it("calls DELETE on resumes/{id}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await resumesService.delete("r1");

      expect(mockApiClient.delete).toHaveBeenCalledWith("resumes/r1");
    });
  });

  describe("generateUploadURL", () => {
    it("calls POST on resumes/upload-url", async () => {
      const mockResponse = {
        upload_url: "https://s3.example.com/upload",
        resume_id: "r3",
      };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await resumesService.generateUploadURL({
        filename: "resume.pdf",
        content_type: "application/pdf",
      });

      expect(mockApiClient.post).toHaveBeenCalledWith("resumes/upload-url", {
        filename: "resume.pdf",
        content_type: "application/pdf",
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("uploadToS3", () => {
    it("sends PUT request to the presigned URL", async () => {
      const mockFetch = vi.fn().mockResolvedValue({ ok: true });
      globalThis.fetch = mockFetch;

      const file = new File(["pdf-content"], "resume.pdf", {
        type: "application/pdf",
      });

      await resumesService.uploadToS3(
        "https://s3.example.com/upload",
        file,
      );

      expect(mockFetch).toHaveBeenCalledWith("https://s3.example.com/upload", {
        method: "PUT",
        headers: { "Content-Type": "application/pdf" },
        body: file,
      });
    });

    it("throws when upload fails", async () => {
      const mockFetch = vi
        .fn()
        .mockResolvedValue({ ok: false, status: 500 });
      globalThis.fetch = mockFetch;

      const file = new File(["content"], "resume.pdf", {
        type: "application/pdf",
      });

      await expect(
        resumesService.uploadToS3("https://s3.example.com/upload", file),
      ).rejects.toThrow("Failed to upload file to S3");
    });
  });

  describe("generateDownloadURL", () => {
    it("calls GET on resumes/{id}/download", async () => {
      const mockResponse = { download_url: "https://s3.example.com/dl" };
      mockApiClient.get.mockResolvedValue(mockResponse);

      const result = await resumesService.generateDownloadURL("r1");

      expect(mockApiClient.get).toHaveBeenCalledWith("resumes/r1/download");
      expect(result).toEqual(mockResponse);
    });
  });

  describe("uploadResume", () => {
    it("orchestrates the full upload flow", async () => {
      mockApiClient.post.mockResolvedValue({
        upload_url: "https://s3.example.com/upload",
        resume_id: "r5",
      });

      const mockFetch = vi.fn().mockResolvedValue({ ok: true });
      globalThis.fetch = mockFetch;

      const mockResume = { id: "r5", title: "resume.pdf" };
      mockApiClient.get.mockResolvedValue(mockResume);

      const file = new File(["content"], "resume.pdf", {
        type: "application/pdf",
      });
      const onProgress = vi.fn();

      const result = await resumesService.uploadResume(file, onProgress);

      expect(mockApiClient.post).toHaveBeenCalledWith("resumes/upload-url", {
        filename: "resume.pdf",
        content_type: "application/pdf",
      });
      expect(mockFetch).toHaveBeenCalled();
      expect(mockApiClient.get).toHaveBeenCalledWith("resumes/r5");
      expect(onProgress).toHaveBeenCalledWith(50);
      expect(onProgress).toHaveBeenCalledWith(100);
      expect(result).toEqual(mockResume);
    });

    it("works without onProgress callback", async () => {
      mockApiClient.post.mockResolvedValue({
        upload_url: "https://s3.example.com/upload",
        resume_id: "r6",
      });
      globalThis.fetch = vi.fn().mockResolvedValue({ ok: true });
      mockApiClient.get.mockResolvedValue({ id: "r6" });

      const file = new File(["content"], "test.pdf", {
        type: "application/pdf",
      });

      const result = await resumesService.uploadResume(file);

      expect(result).toEqual({ id: "r6" });
    });
  });
});
