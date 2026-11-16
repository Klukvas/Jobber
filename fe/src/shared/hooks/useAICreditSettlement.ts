import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/authStore";

function terminalOperations(value: unknown): { id: string; status: string }[] {
  if (!value || typeof value !== "object") return [];
  if ("operations" in value && Array.isArray(value.operations)) return value.operations.flatMap(terminalOperations);
  if ("id" in value && typeof value.id === "string" && "status" in value &&
      typeof value.status === "string" && ["succeeded", "failed", "interrupted"].includes(value.status)) {
    return [{ id: value.id, status: value.status }];
  }
  return [];
}

/** Background interview completion settles the reservation after the POST has returned. */
export function useAICreditSettlement(): void {
  const cache = useQueryClient();
  const userId = useAuthStore((state) => state.user?.id);
  useEffect(() => {
    if (!userId) return;
    const observed = new Map<string, string>();
    return cache.getQueryCache().subscribe((event) => {
      if (event.type !== "updated" || event.action.type !== "success" || event.query.queryKey[0] !== "interview-prep") return;
      let changed = false;
      for (const operation of terminalOperations(event.query.state.data)) {
        if (observed.get(operation.id) === operation.status) continue;
        observed.set(operation.id, operation.status);
        changed = true;
      }
      if (changed) void cache.invalidateQueries({ queryKey: ["subscription"] });
    });
  }, [cache, userId]);
}
