import { useTranslation } from "react-i18next";
import type { PrepOperation, PrepSession } from "@/services/interviewPrepService";
import { Button } from "@/shared/ui/Button";

type Props = {
  operations: readonly PrepOperation[];
  latestVersionCreatedAt?: string;
  isHistoryPage: boolean;
  session?: PrepSession;
  aiAvailable: boolean;
  busy: boolean;
  onRetry: (operation: PrepOperation) => void;
};

const pending = (operation: PrepOperation): boolean =>
  operation.status === "queued" || operation.status === "running";

export function InterviewOperationNotices({
  operations, latestVersionCreatedAt, isHistoryPage, session, aiAvailable, busy, onRetry,
}: Props) {
  const { t, i18n } = useTranslation();
  // Recovered receipts may be appended after the paginated history, so do not
  // rely on array order to identify the most recent plan operation.
  const latestPlan = operations.filter((op) => op.kind === "plan").reduce<PrepOperation | undefined>(
    (latest, op) => !latest || Date.parse(op.created_at) > Date.parse(latest.created_at) ? op : latest,
    undefined,
  );
  const isCurrent = (op: PrepOperation): boolean => {
    if (isHistoryPage) return false;
    if (op.kind === "plan") {
      return op.id === latestPlan?.id && (
        pending(op) || !latestVersionCreatedAt ||
        Date.parse(op.created_at) > Date.parse(latestVersionCreatedAt)
      );
    }
    if (!session || op.session_id !== session.id) return false;
    if (op.kind === "summary") return !session.summary;
    return session.answers.some((answer) => answer.id === op.answer_id && !answer.feedback);
  };
  const notices = operations.filter((op) => op.status !== "succeeded");
  const current = notices.filter(isCurrent);
  const historical = notices.filter((op) => !isCurrent(op));
  const renderNotice = (op: PrepOperation, inHistory: boolean) => (
    <div key={op.id} className="flex flex-wrap items-center justify-between gap-2 rounded-md border p-3 text-sm">
      <p>
        <strong>{t(`jobs.interview.${op.kind}`)}: </strong>
        {t(`jobs.interview.${op.status}`)}
        {inHistory && <> · {new Date(op.created_at).toLocaleString(i18n.language)}</>}
        {op.error_code && <> · {t(`jobs.interview.errors.${op.error_code}`, {
          defaultValue: t("jobs.interview.errors.generic"),
        })}</>}
        {!inHistory && !pending(op) && op.attempts >= 3 && (
          <span className="mt-2 block">
            {t(op.kind === "plan" ? "jobs.interview.exhaustedPlan" : "jobs.interview.exhaustedReview")}
          </span>
        )}
      </p>
      {!inHistory && !pending(op) && op.error_code !== "SOURCE_CHANGED" && op.attempts < 3 && (
        <Button variant="outline" disabled={!aiAvailable || busy} onClick={() => onRetry(op)}>
          {t("jobs.interview.retrySafe")}
        </Button>
      )}
    </div>
  );
  return (
    <>
      {current.map((op) => renderNotice(op, false))}
      {historical.length > 0 && (
        <details className="space-y-2">
          <summary className="min-h-11 cursor-pointer py-3 text-sm font-medium">
            {t("jobs.interview.history")}
          </summary>
          {historical.map((op) => renderNotice(op, true))}
        </details>
      )}
    </>
  );
}
