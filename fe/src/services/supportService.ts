import { apiClient } from "./api";

export interface CreateSupportRequest {
  subject: string;
  message: string;
  page: string;
}

export const supportService = {
  submit: (data: CreateSupportRequest) =>
    apiClient.post<{ message: string }>("support", data),
};
