import type { QueryClient } from "@tanstack/react-query";
import { create } from "zustand";
import { useAuthStore } from "@/stores/authStore";

type AdminScope = {
  actor: string | null;
  session: number;
  revision: number;
  denied: boolean;
};
const actor = () => {
  const auth = useAuthStore.getState();
  return auth.isAuthenticated ? (auth.user?.id ?? null) : null;
};
export const useAdminCapability = create<AdminScope>(() => ({
  actor: actor(),
  session: 0,
  revision: 0,
  denied: false,
}));
const clients = new Map<QueryClient, number>();
const protectedQuery = (key: readonly unknown[]) =>
  key[0] === "admin-users" || key[0] === "admin-user";

function clearProtected(client: QueryClient): void {
  // Cancellation reverts mounted observers before removal; the request fence
  // also discards transports that complete despite cancellation.
  void client.cancelQueries({
    predicate: (query) => protectedQuery(query.queryKey),
  });
  client.removeQueries({
    predicate: (query) => protectedQuery(query.queryKey),
  });
}
export function bindAdminClient(client: QueryClient): () => void {
  clients.set(client, (clients.get(client) ?? 0) + 1);
  return () => {
    const count = clients.get(client) ?? 0;
    if (count <= 1) clients.delete(client);
    else clients.set(client, count - 1);
  };
}
useAuthStore.subscribe(() => {
  const current = useAdminCapability.getState();
  if (current.actor === actor()) return;
  useAdminCapability.setState({
    actor: actor(),
    session: current.session + 1,
    revision: current.revision + 1,
    denied: false,
  });
  clients.forEach((_count, client) => {
    clearProtected(client);
    void client.cancelQueries({ queryKey: ["admin-access"] });
    client.removeQueries({ queryKey: ["admin-access"] });
  });
});
export const adminKeys = {
  access: (scope: AdminScope) =>
    ["admin-access", scope.actor, scope.session] as const,
  users: (scope: AdminScope) =>
    ["admin-users", scope.actor, scope.session] as const,
  detail: (scope: AdminScope, id: string) =>
    ["admin-user", scope.actor, scope.session, id] as const,
};
export function currentAdminRequest(): AdminScope {
  return useAdminCapability.getState();
}
export function isCurrentAdminRequest(request: AdminScope): boolean {
  const current = currentAdminRequest();
  return (
    request.actor === current.actor &&
    request.session === current.session &&
    request.revision === current.revision
  );
}
export function denyAdminRequest(request: AdminScope): void {
  if (!isCurrentAdminRequest(request)) return;
  useAdminCapability.setState({ denied: true, revision: request.revision + 1 });
  clients.forEach((_count, client) => {
    clearProtected(client);
    client.setQueryData(adminKeys.access(request), { allowed: false });
  });
}
export function allowAdminRequest(request: AdminScope): void {
  if (isCurrentAdminRequest(request))
    useAdminCapability.setState({ denied: false });
}
export function assertCurrentAdminRequest(request: AdminScope): void {
  if (!isCurrentAdminRequest(request))
    throw new DOMException("Admin request superseded", "AbortError");
}
