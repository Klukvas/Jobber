import { useEffect, useRef, useState } from "react";
import { useQueries, useQueryClient } from "@tanstack/react-query";
import {
  interviewPrepService,
  type PrepOperation,
} from "@/services/interviewPrepService";

const pending = (operation: PrepOperation) =>
  operation.status === "queued" || operation.status === "running";

// A returned receipt remains addressable even after it leaves the history page.
// Recovery polling is read-only; only the explicit Retry button can enqueue again.
export function usePrepOperationRecovery(job: string, page: PrepOperation[]) {
  const cache = useQueryClient();
  const [receipts, setReceipts] = useState<PrepOperation[]>([]);
  const statuses = useRef(new Map<string, string>());
  const known = new Set(page.map((operation) => operation.id));
  const queries = useQueries({
    queries: receipts.map((receipt) => ({
      queryKey: ["interview-prep", job, "operation", receipt.id],
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        interviewPrepService.operation(job, receipt.id, { signal }),
      enabled: !known.has(receipt.id),
      refetchInterval: (query: {
        state: { data: PrepOperation | undefined };
      }) => (query.state.data && pending(query.state.data) ? 1500 : false),
    })),
  });
  const recovered = receipts.map(
    (receipt, index) =>
      page.find((operation) => operation.id === receipt.id) ??
      queries[index]?.data ??
      receipt,
  );
  const operations = [
    ...page,
    ...recovered.filter((receipt) => !known.has(receipt.id)),
  ];
  useEffect(() => {
    for (const operation of recovered) {
      if (statuses.current.get(operation.id) === operation.status) continue;
      statuses.current.set(operation.id, operation.status);
      if (!pending(operation)) {
        void cache.invalidateQueries({
          queryKey: ["interview-prep", job, "history"],
        });
        if (operation.session_id)
          void cache.invalidateQueries({
            queryKey: ["interview-prep", job, "session", operation.session_id],
          });
      }
    }
  }, [cache, job, recovered]);
  const remember = (operation: PrepOperation) => {
    cache.setQueryData(
      ["interview-prep", job, "operation", operation.id],
      operation,
    );
    setReceipts((current) =>
      [
        operation,
        ...current.filter((receipt) => receipt.id !== operation.id),
      ].slice(0, 30),
    );
  };
  return { operations, remember };
}
