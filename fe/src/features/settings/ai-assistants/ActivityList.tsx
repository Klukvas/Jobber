import type { UseQueryResult } from "@tanstack/react-query";
import { formatDistance } from "date-fns";
import { useTranslation } from "react-i18next";
import type { AssistantActivity } from "@/services/apiTokensService";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { ErrorState } from "@/shared/ui/ErrorState";
import { ListLoading } from "./ListState";
import { useNow } from "./useNow";

const errorLabels: Record<string, string> = {
  PAID_FEATURE: "paid",
  VALIDATION_ERROR: "validation",
  INVALID_ARGUMENT: "validation",
  NOT_FOUND: "notFound",
  UNAUTHORIZED: "unauthorized",
  FORBIDDEN: "unauthorized",
  RATE_LIMITED: "rateLimit",
  RATE_LIMIT_EXCEEDED: "rateLimit",
  LIMIT_REACHED: "limit",
  CREDIT_UNAVAILABLE: "credits",
  AI_BUDGET_REACHED: "budget",
  AI_OPERATION_CONFLICT: "conflict",
  INTERNAL_ERROR: "internal",
  TOOL_ERROR: "internal",
};

export function ActivityList({ query }: { query: UseQueryResult<AssistantActivity[], Error> }) {
  const { t } = useTranslation();
  const locale = useDateLocale();
  const now = useNow();
  const calls = query.data ?? [];
  return (
    <section className="space-y-3" aria-labelledby="mcp-activity-title">
      <h3 id="mcp-activity-title" className="font-semibold">{t("settings.aiAssistants.activity")}</h3>
      {query.isError && (
        <ErrorState message={t("settings.aiAssistants.activityError")} onRetry={() => void query.refetch()} />
      )}
      {query.isPending ? <ListLoading kind="activity" /> : calls.length ? (
        <ul className="divide-y rounded-lg border">
          {calls.map((call) => {
            const errorKey = errorLabels[call.error_code?.toUpperCase() ?? ""] ?? "unknown";
            const result = call.ok ? t("settings.aiAssistants.success")
              : `${t("settings.aiAssistants.error")}: ${t(`settings.aiAssistants.errors.${errorKey}`)}`;
            return (
              <li key={call.id} className="space-y-2 p-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <code className="break-all text-sm">{call.tool}</code>
                  <span className={`rounded border px-2 py-1 text-xs ${call.ok ? "text-green-700 dark:text-green-400" : "text-destructive"}`}>
                    {result}
                  </span>
                </div>
                <div className="flex flex-wrap justify-between gap-2 text-xs text-muted-foreground">
                  <span className="break-all">{call.token_name ?? t("settings.aiAssistants.deletedToken")}</span>
                  <time dateTime={call.created_at} className="tabular-nums">
                    {formatDistance(new Date(call.created_at), now, { addSuffix: true, locale })}
                  </time>
                </div>
              </li>
            );
          })}
        </ul>
      ) : !query.isError && (
        <p className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
          {t("settings.aiAssistants.activityEmpty")}
        </p>
      )}
    </section>
  );
}
