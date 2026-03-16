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

import { coverLetterService } from "../coverLetterService";

describe("coverLetterService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("list calls GET on cover-letters", async () => {
    const mockData = [{ id: "cl-1", title: "Letter 1" }];
    mockApiClient.get.mockResolvedValue(mockData);

    const result = await coverLetterService.list();

    expect(mockApiClient.get).toHaveBeenCalledWith("cover-letters");
    expect(result).toEqual(mockData);
  });

  it("create passes data to POST on cover-letters", async () => {
    const input = { title: "New Letter", template: "professional" };
    const mockResponse = { id: "cl-2", ...input };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await coverLetterService.create(input as never);

    expect(mockApiClient.post).toHaveBeenCalledWith("cover-letters", input);
    expect(result).toEqual(mockResponse);
  });

  it("duplicate calls POST on cover-letters/{id}/duplicate", async () => {
    const mockResponse = { id: "cl-3", title: "Letter 1 (Copy)" };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await coverLetterService.duplicate("cl-1");

    expect(mockApiClient.post).toHaveBeenCalledWith(
      "cover-letters/cl-1/duplicate",
    );
    expect(result).toEqual(mockResponse);
  });
});
