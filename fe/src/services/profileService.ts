import { apiClient, type RequestOptions } from "./api";
import type { UserDTO } from "@/shared/types/api";

/**
 * The signed-in person's own account. There is no user id in the call: the
 * server takes it from the access token, so a client can only ever address
 * itself.
 *
 * Read-only, and the API is too — `GET /profile` is the whole surface. The
 * store the rest of the app reads the account from is written at sign-in and
 * never refreshed, so a screen that wants the row as it actually stands has to
 * ask for it; nothing here writes one back.
 */
export const profileService = {
  async get(options?: RequestOptions): Promise<UserDTO> {
    return apiClient.get<UserDTO>("profile", options);
  },
};
