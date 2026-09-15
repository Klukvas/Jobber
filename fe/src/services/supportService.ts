import { apiClient } from "./api";

export interface CreateSupportRequest {
  subject: string;
  message: string;
  page: string;
}

export const supportService = {
  submit: (data: CreateSupportRequest) =>
    apiClient.post<{ message: string }>("support", data),

  /**
   * Whether this deployment has a support channel wired up. The form is hidden
   * when it does not, rather than letting a customer write a message that has
   * nowhere to go.
   */
  status: () => apiClient.get<{ available: boolean }>("support/status"),
};
