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

  it("update calls PATCH on resume-builder/{id}", async () => {
    const data = { title: "Renamed" };
    mockApiClient.patch.mockResolvedValue({ id: "r1", ...data });

    const result = await resumeBuilderService.update("r1", data as never);

    expect(mockApiClient.patch).toHaveBeenCalledWith("resume-builder/r1", data);
    expect(result).toEqual({ id: "r1", title: "Renamed" });
  });

  it("delete calls DELETE on resume-builder/{id}", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);

    await resumeBuilderService.delete("r1");

    expect(mockApiClient.delete).toHaveBeenCalledWith("resume-builder/r1");
  });

  it("upsertContact calls PUT on resume-builder/{id}/contact", async () => {
    const data = { email: "me@example.com" };
    mockApiClient.put.mockResolvedValue({ id: "ct-1", ...data });

    const result = await resumeBuilderService.upsertContact(
      "r1",
      data as never,
    );

    expect(mockApiClient.put).toHaveBeenCalledWith(
      "resume-builder/r1/contact",
      data,
    );
    expect(result).toEqual({ id: "ct-1", email: "me@example.com" });
  });

  it("upsertSummary calls PUT on resume-builder/{id}/summary", async () => {
    const data = { text: "A summary" };
    mockApiClient.put.mockResolvedValue({ id: "sm-1", ...data });

    await resumeBuilderService.upsertSummary("r1", data as never);

    expect(mockApiClient.put).toHaveBeenCalledWith(
      "resume-builder/r1/summary",
      data,
    );
  });

  it("updateSectionOrder calls PUT on resume-builder/{id}/section-order", async () => {
    const data = { order: [{ section: "experiences", position: 0 }] };
    mockApiClient.put.mockResolvedValue([]);

    await resumeBuilderService.updateSectionOrder("r1", data as never);

    expect(mockApiClient.put).toHaveBeenCalledWith(
      "resume-builder/r1/section-order",
      data,
    );
  });

  describe("typed section methods delegate to the right endpoints", () => {
    beforeEach(() => {
      mockApiClient.post.mockResolvedValue({ id: "x" });
      mockApiClient.patch.mockResolvedValue({ id: "x" });
      mockApiClient.delete.mockResolvedValue(undefined);
    });

    it.each([
      ["createExperience", "experiences"],
      ["createEducation", "educations"],
      ["createSkill", "skills"],
      ["createLanguage", "languages"],
      ["createCertification", "certifications"],
      ["createProject", "projects"],
      ["createVolunteering", "volunteering"],
      ["createCustomSection", "custom-sections"],
    ])("%s POSTs to the %s collection", async (method, section) => {
      const data = { foo: "bar" };
      await (
        resumeBuilderService as unknown as Record<
          string,
          (id: string, d: unknown) => Promise<unknown>
        >
      )[method]("r1", data);
      expect(mockApiClient.post).toHaveBeenCalledWith(
        `resume-builder/r1/${section}`,
        data,
      );
    });

    it.each([
      ["updateExperience", "experiences"],
      ["updateEducation", "educations"],
      ["updateSkill", "skills"],
      ["updateLanguage", "languages"],
      ["updateCertification", "certifications"],
      ["updateProject", "projects"],
      ["updateVolunteering", "volunteering"],
      ["updateCustomSection", "custom-sections"],
    ])("%s PATCHes the %s entry", async (method, section) => {
      const data = { foo: "baz" };
      await (
        resumeBuilderService as unknown as Record<
          string,
          (id: string, e: string, d: unknown) => Promise<unknown>
        >
      )[method]("r1", "e1", data);
      expect(mockApiClient.patch).toHaveBeenCalledWith(
        `resume-builder/r1/${section}/e1`,
        data,
      );
    });

    it.each([
      ["deleteExperience", "experiences"],
      ["deleteEducation", "educations"],
      ["deleteSkill", "skills"],
      ["deleteLanguage", "languages"],
      ["deleteCertification", "certifications"],
      ["deleteProject", "projects"],
      ["deleteVolunteering", "volunteering"],
      ["deleteCustomSection", "custom-sections"],
    ])("%s DELETEs the %s entry", async (method, section) => {
      await (
        resumeBuilderService as unknown as Record<
          string,
          (id: string, e: string) => Promise<void>
        >
      )[method]("r1", "e1");
      expect(mockApiClient.delete).toHaveBeenCalledWith(
        `resume-builder/r1/${section}/e1`,
      );
    });
  });
});
