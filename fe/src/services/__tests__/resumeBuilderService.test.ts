import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  postFormData: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { resumeBuilderService } from "../resumeBuilderService";

describe("resumeBuilderService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("list calls GET on resume-builder", async () => {
    const mockData = [{ id: "r1", title: "Resume 1" }];
    mockApiClient.get.mockResolvedValue(mockData);

    const result = await resumeBuilderService.list();

    expect(mockApiClient.get).toHaveBeenCalledWith("resume-builder");
    expect(result).toEqual(mockData);
  });

  it("getById calls GET on resume-builder/{id}", async () => {
    const mockResume = { id: "r1", title: "Resume 1" };
    mockApiClient.get.mockResolvedValue(mockResume);

    const result = await resumeBuilderService.getById("r1");

    expect(mockApiClient.get).toHaveBeenCalledWith("resume-builder/r1");
    expect(result).toEqual(mockResume);
  });

  it("create passes data to POST on resume-builder", async () => {
    const input = { title: "New Resume", template: "modern" };
    const mockResponse = { id: "r2", ...input };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.create(input as never);

    expect(mockApiClient.post).toHaveBeenCalledWith("resume-builder", input);
    expect(result).toEqual(mockResponse);
  });

  it("duplicate calls POST on resume-builder/{id}/duplicate", async () => {
    const mockResponse = { id: "r3", title: "Resume 1 (Copy)" };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.duplicate("r1");

    expect(mockApiClient.post).toHaveBeenCalledWith(
      "resume-builder/r1/duplicate",
    );
    expect(result).toEqual(mockResponse);
  });

  it("createSection delegates to POST on resume-builder/{id}/{section}", async () => {
    const data = { company: "Acme", role: "Engineer" };
    const mockResponse = { id: "exp-1", ...data };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.createSection(
      "r1",
      "experiences",
      data,
    );

    expect(mockApiClient.post).toHaveBeenCalledWith(
      "resume-builder/r1/experiences",
      data,
    );
    expect(result).toEqual(mockResponse);
  });

  it("updateSection delegates to PATCH on resume-builder/{id}/{section}/{entryId}", async () => {
    const data = { company: "Updated Corp" };
    const mockResponse = { id: "exp-1", ...data };
    mockApiClient.patch.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.updateSection(
      "r1",
      "experiences",
      "exp-1",
      data,
    );

    expect(mockApiClient.patch).toHaveBeenCalledWith(
      "resume-builder/r1/experiences/exp-1",
      data,
    );
    expect(result).toEqual(mockResponse);
  });

  it("deleteSection delegates to DELETE on resume-builder/{id}/{section}/{entryId}", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);

    await resumeBuilderService.deleteSection("r1", "skills", "sk-1");

    expect(mockApiClient.delete).toHaveBeenCalledWith(
      "resume-builder/r1/skills/sk-1",
    );
  });

  it("importFromText passes data to POST on resume-builder/import/text", async () => {
    const data = { text: "My resume content", title: "Imported" };
    const mockResponse = { id: "r4", title: "Imported" };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.importFromText(data);

    expect(mockApiClient.post).toHaveBeenCalledWith(
      "resume-builder/import/text",
      data,
    );
    expect(result).toEqual(mockResponse);
  });

  it("importFromPDF constructs FormData with file", async () => {
    const file = new File(["pdf-content"], "resume.pdf", {
      type: "application/pdf",
    });
    const mockResponse = { id: "r5", title: "resume.pdf" };
    mockApiClient.postFormData.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.importFromPDF(file);

    expect(mockApiClient.postFormData).toHaveBeenCalledWith(
      "resume-builder/import/pdf",
      expect.any(FormData),
    );

    const sentFormData = mockApiClient.postFormData.mock
      .calls[0][1] as FormData;
    expect(sentFormData.get("file")).toBe(file);
    expect(sentFormData.has("title")).toBe(false);
    expect(result).toEqual(mockResponse);
  });

  it("importFromPDF includes title in FormData when provided", async () => {
    const file = new File(["pdf-content"], "resume.pdf", {
      type: "application/pdf",
    });
    const mockResponse = { id: "r6", title: "Custom Title" };
    mockApiClient.postFormData.mockResolvedValue(mockResponse);

    const result = await resumeBuilderService.importFromPDF(
      file,
      "Custom Title",
    );

    expect(mockApiClient.postFormData).toHaveBeenCalledWith(
      "resume-builder/import/pdf",
      expect.any(FormData),
    );

    const sentFormData = mockApiClient.postFormData.mock
      .calls[0][1] as FormData;
    expect(sentFormData.get("file")).toBe(file);
    expect(sentFormData.get("title")).toBe("Custom Title");
    expect(result).toEqual(mockResponse);
  });
});
