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

import { remindersService } from "../remindersService";

describe("remindersService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("lists reminders for a job", async () => {
    const data = [{ id: "r1" }];
    mockApiClient.get.mockResolvedValue(data);

    const result = await remindersService.listByJob("job-1");

    expect(mockApiClient.get).toHaveBeenCalledWith("jobs/job-1/reminders");
    expect(result).toEqual(data);
  });

  it("creates a reminder", async () => {
    const input = {
      job_id: "job-1",
      message: "ping",
      remind_at: "2026-08-12T14:30:00.000Z",
    };
    const created = { id: "r1", ...input, is_done: false };
    mockApiClient.post.mockResolvedValue(created);

    const result = await remindersService.create(input);

    expect(mockApiClient.post).toHaveBeenCalledWith("reminders", input);
    expect(result).toEqual(created);
  });

  it("updates a reminder", async () => {
    const updated = { id: "r1", is_done: true };
    mockApiClient.patch.mockResolvedValue(updated);

    const result = await remindersService.update("r1", { is_done: true });

    expect(mockApiClient.patch).toHaveBeenCalledWith("reminders/r1", {
      is_done: true,
    });
    expect(result).toEqual(updated);
  });

  it("deletes a reminder", async () => {
    mockApiClient.delete.mockResolvedValue(undefined);

    await remindersService.delete("r1");

    expect(mockApiClient.delete).toHaveBeenCalledWith("reminders/r1");
  });
});
