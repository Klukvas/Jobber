import { apiClient } from "./api";
import type {
  ReminderDTO,
  CreateReminderRequest,
  UpdateReminderRequest,
} from "@/shared/types/api";

export const remindersService = {
  async listByJob(jobId: string): Promise<ReminderDTO[]> {
    return apiClient.get<ReminderDTO[]>(`jobs/${jobId}/reminders`);
  },

  async create(data: CreateReminderRequest): Promise<ReminderDTO> {
    return apiClient.post<ReminderDTO>("reminders", data);
  },

  async update(
    id: string,
    data: UpdateReminderRequest,
  ): Promise<ReminderDTO> {
    return apiClient.patch<ReminderDTO>(`reminders/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`reminders/${id}`);
  },
};
