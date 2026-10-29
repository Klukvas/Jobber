import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { adminService } from "@/services/adminService";
import {
  adminKeys,
  bindAdminClient,
  useAdminCapability,
} from "@/shared/lib/adminCapability";

export function useAdminScope() {
  const client = useQueryClient();
  const scope = useAdminCapability();
  useEffect(() => bindAdminClient(client), [client]);
  return scope;
}

export function useAdminAccess() {
  const scope = useAdminScope();
  const query = useQuery({
    queryKey: adminKeys.access(scope),
    queryFn: ({ signal }) => adminService.access(signal),
    enabled: !!scope.actor,
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchInterval: 30_000,
    refetchOnWindowFocus: "always",
  });
  return scope.denied || !scope.actor
    ? { ...query, data: { allowed: false }, isPending: false }
    : query;
}
