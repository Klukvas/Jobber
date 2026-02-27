import { apiClient } from "./api";
import type {
  CalendarStatusDTO,
  CalendarEventDTO,
  CreateCalendarEventRequest,
  OAuthURLResponse,
} from "@/shared/types/api";

export const calendarService = {
  async getAuthURL(): Promise<OAuthURLResponse> {
    return apiClient.get<OAuthURLResponse>("calendar/auth");
  },

  async getStatus(): Promise<CalendarStatusDTO> {
    return apiClient.get<CalendarStatusDTO>("calendar/status");
  },

  async disconnect(): Promise<void> {
    return apiClient.delete<void>("calendar");
  },

  async createEvent(
    data: CreateCalendarEventRequest,
  ): Promise<CalendarEventDTO> {
    return apiClient.post<CalendarEventDTO>("calendar/events", data);
  },

  async deleteEvent(stageId: string): Promise<void> {
    return apiClient.delete<void>(`calendar/events/${stageId}`);
  },
};
