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

import { contentLibraryService } from "../contentLibraryService";

describe("contentLibraryService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("list calls GET on content-library", async () => {
    const mockData = [{ id: "entry-1", title: "Snippet 1" }];
    mockApiClient.get.mockResolvedValue(mockData);

    const result = await contentLibraryService.list();

    expect(mockApiClient.get).toHaveBeenCalledWith("content-library");
    expect(result).toEqual(mockData);
  });

  it("create passes data to POST on content-library", async () => {
    const input = { title: "New Snippet", content: "Some content" };
    const mockResponse = { id: "entry-2", ...input };
    mockApiClient.post.mockResolvedValue(mockResponse);

    const result = await contentLibraryService.create(input as never);

    expect(mockApiClient.post).toHaveBeenCalledWith("content-library", input);
    expect(result).toEqual(mockResponse);
  });

  it("remove calls DELETE on content-library/{id}", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);

    await contentLibraryService.remove("entry-1");

    expect(mockApiClient.delete).toHaveBeenCalledWith(
      "content-library/entry-1",
    );
  });
});
