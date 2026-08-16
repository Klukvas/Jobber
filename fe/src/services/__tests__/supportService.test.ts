import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  post: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { supportService } from "../supportService";

describe("supportService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("posts the support request to the support endpoint", async () => {
    const data = {
      subject: "Help me",
      message: "Something is broken",
      page: "/app/jobs",
    };
    const response = { message: "received" };
    mockApiClient.post.mockResolvedValue(response);

    const result = await supportService.submit(data);

    expect(mockApiClient.post).toHaveBeenCalledWith("support", data);
    expect(result).toEqual(response);
  });

  it("propagates errors from the API client", async () => {
    mockApiClient.post.mockRejectedValue(new Error("boom"));

    await expect(
      supportService.submit({ subject: "s", message: "m", page: "/p" }),
    ).rejects.toThrow("boom");
  });
});
