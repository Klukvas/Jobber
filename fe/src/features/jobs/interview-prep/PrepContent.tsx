import { useTranslation } from "react-i18next";
import type {
  PrepEvidence,
  PrepQuestion,
  PrepReview,
  PrepVersion,
} from "@/services/interviewPrepService";
import { Button } from "@/shared/ui/Button";

export function Evidence({ items }: { items: readonly PrepEvidence[] }) {
  const { t } = useTranslation();
  return items.length ? (
    <ul className="space-y-1 border-l-2 pl-3 text-sm text-muted-foreground">
      {items.map((e, i) => (
        <li key={`${e.fact_id}:${i}`}>
          <q className="whitespace-pre-wrap break-words">{e.quote}</q>
        </li>
      ))}
    </ul>
  ) : (
    <p className="text-sm text-muted-foreground">
      {t("jobs.interview.clarify")}
    </p>
  );
}
export function QuestionContent({ question }: { question: PrepQuestion }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-3 break-words">
      <p className="text-xs text-muted-foreground">
        {t(`jobs.interview.${question.kind}`)} ·{" "}
        {t(`jobs.interview.${question.difficulty}`)}
      </p>
      <h4 className="font-medium">{question.question}</h4>
      <p className="text-sm">
        <strong>{t("jobs.interview.purpose")}: </strong>
        {question.purpose}
      </p>
      <p className="text-sm whitespace-pre-wrap">
        <strong>{t("jobs.interview.advice")}: </strong>
        {question.advice}
      </p>
      <Evidence items={question.evidence} />
      {question.star && (
        <div className="space-y-2 rounded-md bg-muted p-3 text-sm">
          <p className="font-medium">{t("jobs.interview.star")}</p>
          {(["situation", "task", "action", "result"] as const).map((key) => (
            <p key={key}>
              <strong>{t(`jobs.interview.${key}`)}: </strong>
              {question.star?.[key]}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
export function PlanContent({
  version,
  busy,
  onStudied,
}: {
  version: PrepVersion;
  busy: boolean;
  onStudied: (id: string, studied: boolean) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div>
        <h3 className="mb-3 font-semibold">
          {t("jobs.interview.requirements")}
        </h3>
        <div className="space-y-3">
          {version.plan.requirements.map((r, i) => (
            <div key={i} className="space-y-2 rounded-lg border p-4">
              <h4 className="font-medium">{r.requirement}</h4>
              <p className="text-sm">{r.assessment}</p>
              <Evidence items={r.evidence} />
            </div>
          ))}
        </div>
      </div>
      <div>
        <h3 className="mb-3 font-semibold">
          {t("jobs.interview.questions")} · {version.studied.length}/
          {version.plan.questions.length}
        </h3>
        <div className="space-y-3">
          {version.plan.questions.map((q) => (
            <article key={q.id} className="space-y-3 rounded-lg border p-4">
              <QuestionContent question={q} />
              <Button
                variant="outline"
                disabled={busy}
                aria-pressed={version.studied.includes(q.id)}
                onClick={() => onStudied(q.id, !version.studied.includes(q.id))}
              >
                {version.studied.includes(q.id)
                  ? t("jobs.interview.studied")
                  : t("jobs.interview.markStudied")}
              </Button>
            </article>
          ))}
        </div>
      </div>
      <div>
        <h3 className="mb-3 font-semibold">{t("jobs.interview.gaps")}</h3>
        {version.plan.gaps.map((g, i) => (
          <p key={i} className="mb-3 rounded-lg border p-4 text-sm">
            <strong>
              {t(`jobs.interview.${g.priority}`)} · {g.gap}
            </strong>
            <br />
            {g.action}
          </p>
        ))}
      </div>
      <div>
        <h3 className="mb-3 font-semibold">
          {t("jobs.interview.askInterviewer")}
        </h3>
        <ul className="list-disc space-y-2 pl-5 text-sm">
          {version.plan.interviewer_questions.map((q, i) => (
            <li key={i}>{q}</li>
          ))}
        </ul>
      </div>
    </div>
  );
}
export function ReviewContent({ review }: { review: PrepReview }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-4 break-words">
      <p className="text-xs text-muted-foreground">
        {t("jobs.interview.scale")}
      </p>
      {review.criteria.map((criterion) => (
        <div key={criterion.name} className="space-y-2 rounded-md border p-3">
          <h4 className="font-medium">
            {t(`jobs.interview.${criterion.name}`)} · {criterion.score}/4
          </h4>
          <p className="text-sm">{criterion.explanation}</p>
          <Evidence items={criterion.evidence} />
        </div>
      ))}
      {(["strengths", "improvements", "revisit"] as const).map((key) => (
        <div key={key}>
          <h4 className="font-medium">{t(`jobs.interview.${key}`)}</h4>
          <ul className="list-disc space-y-1 pl-5 text-sm">
            {review[key].map((text, i) => (
              <li key={i}>{text}</li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
