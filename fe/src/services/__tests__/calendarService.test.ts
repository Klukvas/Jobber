import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiClient: mockApiClient,
}));

import { calendarService } from "../calendarService";

describe("calendarService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getAuthURL", () => {
    it("calls GET on calendar/auth", async () => {
      const mockResponse = { url: "https://accounts.google.com/..." };
      mockApiClient.get.mockResolvedValue(mockResponse);

      const result = await calendarService.getAuthURL();

      expect(mockApiClient.get).toHaveBeenCalledWith("calendar/auth");
      expect(result).toEqual(mockResponse);
    });
  });

  describe("getStatus", () => {
    it("calls GET on calendar/status", async () => {
      const mockStatus = { connected: true };
      mockApiClient.get.mockResolvedValue(mockStatus);

      const result = await calendarService.getStatus();

      expect(mockApiClient.get).toHaveBeenCalledWith("calendar/status");
      expect(result).toEqual(mockStatus);
    });
  });

  describe("disconnect", () => {
    it("calls DELETE on calendar", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await calendarService.disconnect();

      expect(mockApiClient.delete).toHaveBeenCalledWith("calendar");
    });
  });

  describe("createEvent", () => {
    it("calls POST on calendar/events with data", async () => {
      const input = { stage_id: "s1", title: "Interview" };
      const mockResponse = { id: "e1", ...input };
      mockApiClient.post.mockResolvedValue(mockResponse);

      const result = await calendarService.createEvent(input as never);

      expect(mockApiClient.post).toHaveBeenCalledWith(
        "calendar/events",
        input,
      );
      expect(result).toEqual(mockResponse);
    });
  });

  describe("deleteEvent", () => {
    it("calls DELETE on calendar/events/{stageId}", async () => {
      mockApiClient.delete.mockResolvedValue(undefined);

      await calendarService.deleteEvent("s1");

      expect(mockApiClient.delete).toHaveBeenCalledWith(
        "calendar/events/s1",
      );
    });
  });
});
