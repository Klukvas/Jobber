import { useCallback } from "react";
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { apiClient, ApiError } from "@/services/api";

export function useAIAvailability(enabled = true) {
  const queryClient = useQueryClient();
  const markUnavailable = useCallback((error: unknown) => markAIUnavailable(queryClient, error), [queryClient]);
  const query = useQuery({
    queryKey: ["features", "ai"],
    queryFn: ({ signal }) => apiClient.get<{ available: boolean }>("features/ai", { signal }),
    enabled,
    staleTime: 30_000,
    retry: false,
  });
  // A failed probe is unknown, never evidence of known unavailability.
  return { unavailable: query.data?.available === false, isChecking: query.isLoading, markUnavailable };
}

export function isAIUnavailable(error: unknown): boolean {
  return error instanceof ApiError && (
    ["AI_NOT_CONFIGURED", "AI_UNAVAILABLE"].includes(error.code) ||
    (error.status === 404 && error.unstructuredResponse)
  );
}

export function markAIUnavailable(client: QueryClient, error: unknown): void {
  if (isAIUnavailable(error)) client.setQueryData(["features", "ai"], { available: false });
}

export function aiErrorKey(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.code === "CREDIT_UNAVAILABLE") return "credits.exhausted";
    if (error.code === "AI_BUDGET_REACHED") return "credits.budgetReached";
    if (error.code === "AI_OPERATION_CONFLICT") return "credits.operationConflict";
  }
  return isAIUnavailable(error) ? "ai.unavailable" : "ai.requestFailed";
}
