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

import { tagsService } from "../tagsService";

describe("tagsService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("lists all tags", async () => {
    mockApiClient.get.mockResolvedValue([{ id: "t1" }]);
    const result = await tagsService.list();
    expect(mockApiClient.get).toHaveBeenCalledWith("tags");
    expect(result).toEqual([{ id: "t1" }]);
  });

  it("creates a tag", async () => {
    const input = { name: "urgent", color: "#2563eb" };
    mockApiClient.post.mockResolvedValue({ id: "t1", ...input });
    const result = await tagsService.create(input);
    expect(mockApiClient.post).toHaveBeenCalledWith("tags", input);
    expect(result).toEqual({ id: "t1", ...input });
  });

  it("attaches a tag to a job", async () => {
    mockApiClient.post.mockResolvedValue(undefined);
    await tagsService.attach("t1", { entity_type: "job", entity_id: "job-1" });
    expect(mockApiClient.post).toHaveBeenCalledWith("tags/t1/relations", {
      entity_type: "job",
      entity_id: "job-1",
    });
  });

  it("detaches a tag using query params", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);
    await tagsService.detach("t1", "job", "job-1");
    const url = mockApiClient.delete.mock.calls[0][0] as string;
    expect(url).toContain("tags/t1/relations?");
    expect(url).toContain("entity_type=job");
    expect(url).toContain("entity_id=job-1");
  });

  it("lists tags for a job", async () => {
    mockApiClient.get.mockResolvedValue([]);
    await tagsService.listByJob("job-1");
    expect(mockApiClient.get).toHaveBeenCalledWith("jobs/job-1/tags");
  });

  it("deletes a tag", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);
    await tagsService.delete("t1");
    expect(mockApiClient.delete).toHaveBeenCalledWith("tags/t1");
  });
});
