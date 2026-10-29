import { apiClient, ApiError } from "./api";
import {
  allowAdminRequest,
  assertCurrentAdminRequest,
  currentAdminRequest,
  denyAdminRequest,
} from "@/shared/lib/adminCapability";

async function request<T>(
  operation: () => Promise<T>,
  access = false,
): Promise<T> {
  const scope = currentAdminRequest();
  if (!access && scope.denied)
    throw new DOMException("Admin access denied", "AbortError");
  try {
    const result = await operation();
    assertCurrentAdminRequest(scope);
    if (access) {
      if ((result as { allowed: boolean }).allowed) allowAdminRequest(scope);
      else denyAdminRequest(scope);
    }
    return result;
  } catch (error) {
    // apiClient has already completed its normal refresh/replay by this point.
    if (
      error instanceof ApiError &&
      (error.status === 401 || error.status === 403)
    ) {
      denyAdminRequest(scope);
    }
    throw error;
  }
}

export type AdminUser = {
  id: string;
  email: string;
  name: string;
  locale: string;
  email_verified: boolean;
  created_at: string;
  provider_plan: string;
  provider_status: string;
  manual_until: string | null;
};
export type ManualGrant = {
  id: string;
  period_start: string;
  period_end: string;
  revoked_at: string | null;
  actor_id: string;
  reason: string;
  revoked_by: string | null;
  revoke_reason: string | null;
  credits: number;
  used: number;
  active: boolean;
};
export type AdminDetail = {
  user: AdminUser;
  provider_subscription_id: string | null;
  provider_account_id: string | null;
  provider_period_end: string | null;
  provider_cancel_at: string | null;
  effective_plan: string;
  manual_grants: ManualGrant[];
};
export const adminService = {
  access: (signal?: AbortSignal) =>
    request(
      () => apiClient.get<{ allowed: boolean }>("admin/access", { signal }),
      true,
    ),
  users: (search: string, page: number, signal?: AbortSignal) =>
    request(() =>
      apiClient.get<{
        users: AdminUser[];
        total: number;
        page: number;
        limit: number;
      }>(
        `admin/users?${new URLSearchParams({ search, page: String(page), limit: "20" })}`,
        { signal },
      ),
    ),
  detail: (id: string, signal?: AbortSignal) =>
    request(() => apiClient.get<AdminDetail>(`admin/users/${id}`, { signal })),
  update: (id: string, profile: { name: string; locale: string }) =>
    request(() => apiClient.patch(`admin/users/${id}`, profile)),
  grant: (id: string, grant: { id: string; days: number; reason: string }) =>
    request(() => apiClient.post(`admin/users/${id}/manual-access`, grant)),
  revoke: (id: string, grant: string, reason: string) =>
    request(() =>
      apiClient.post(`admin/users/${id}/manual-access/${grant}/revoke`, {
        reason,
      }),
    ),
};
