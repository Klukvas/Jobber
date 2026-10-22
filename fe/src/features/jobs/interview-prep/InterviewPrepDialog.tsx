import { usePrepOperationRecovery } from "./usePrepOperationRecovery";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { ApiError } from "@/services/api";
import {
  interviewPrepService as service,
  interviewTextLength,
  type PrepLocale,
  type PrepOperation,
  type PrepSession,
} from "@/services/interviewPrepService";
import { Button } from "@/shared/ui/Button";
import { Select } from "@/shared/ui/Select";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/shared/ui/Dialog";
import type { InterviewProps } from "./JobInterviewPrepCard";
import { PlanContent, QuestionContent, ReviewContent } from "./PrepContent";

const pending = (operation: PrepOperation): boolean =>
  operation.status === "queued" || operation.status === "running";
function errorCode(error: unknown): string {
  return error instanceof ApiError ? error.code : "INTERNAL_ERROR";
}

export function InterviewPrepDialog({
  job,
  builderResumes,
  isLoading,
  isError,
  isJobDirty,
  isJobSaving,
  onClose,
}: InterviewProps & { onClose: () => void }) {
  const { t, i18n } = useTranslation();
  const cache = useQueryClient();
  const key = ["interview-prep", job.id];
  const [resumeChoice, setResume] = useState<string | null>(null);
  const resume =
    resumeChoice ??
    (job.resume?.type === "builder" &&
    builderResumes.some((r) => r.id === job.resume?.id)
      ? job.resume.id
      : (builderResumes[0]?.id ?? ""));
  const [locale, setLocale] = useState<PrepLocale>(
    i18n.language.startsWith("ru")
      ? "ru"
      : i18n.language.startsWith("uk")
        ? "uk"
        : "en",
  );
  const [offset, setOffset] = useState(0);
  const [selectedVersion, setSelectedVersion] = useState("");
  const [selectedSession, setSelectedSession] = useState("");
  const [tab, setTab] = useState<"plan" | "practice">("plan");
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [latestOperation, setLatestOperation] = useState("");
  const [notice, setNotice] = useState("");
  const [requestError, setRequestError] = useState("");
  const requestKeys = useRef(new Map<string, string>());
  const questionHeading = useRef<HTMLDivElement>(null);
  const requestKey = (action: string): string => {
    const existing = requestKeys.current.get(action);
    if (existing) return existing;
    const id = crypto.randomUUID();
    requestKeys.current.set(action, id);
    return id;
  };
  const history = useQuery({
    queryKey: [...key, "history", offset],
    queryFn: ({ signal }) => service.history(job.id, offset, { signal }),
    refetchInterval: (query) =>
      query.state.data?.operations.some(pending) ? 1500 : 15000,
  });
  const { operations, remember } = usePrepOperationRecovery(
    job.id,
    history.data?.operations ?? [],
  );
  const latest = operations.find((op) => op.id === latestOperation);
  const versionId =
    selectedVersion ||
    latest?.version_id ||
    history.data?.versions[0]?.id ||
    "";
  const versionQuery = useQuery({
    queryKey: [...key, "version", versionId],
    queryFn: ({ signal }) => service.version(job.id, versionId, { signal }),
    enabled: !!versionId,
    refetchInterval: 15000,
  });
  const sessionId = selectedSession;
  const sessionQuery = useQuery({
    queryKey: [...key, "session", sessionId],
    queryFn: ({ signal }) => service.session(job.id, sessionId, { signal }),
    enabled: !!sessionId,
    refetchInterval: operations.some(
      (op) => op.session_id === sessionId && pending(op),
    )
      ? 1500
      : 15000,
  });
  const version = versionQuery.data;
  const session = sessionQuery.data;
  const aiAvailable = history.data?.ai_available ?? false;
  const sessionVersion = useQuery({
    queryKey: [...key, "version", session?.version_id ?? ""],
    queryFn: ({ signal }) =>
      service.version(job.id, session?.version_id ?? "", { signal }),
    enabled: !!session && session.version_id !== versionId,
    refetchInterval: 15000,
  });
  const practiceStale =
    version && session?.version_id === version.id
      ? version.stale
      : sessionVersion.data?.stale;
  const next = session?.questions.find(
    (q) => !session.answers.some((a) => a.question_id === q.id),
  );
  const draftKey = session && next ? `${session.id}:${next.id}` : "";
  const answer = drafts[draftKey] ?? "";
  const setAnswer = (value: string) => {
    if (draftKey) setDrafts((current) => ({ ...current, [draftKey]: value }));
  };
  useEffect(() => {
    questionHeading.current?.focus();
  }, [sessionId, next?.id]);
  const refresh = async () => {
    await cache.invalidateQueries({ queryKey: key });
    const saved = cache.getQueryData<PrepSession>([
      ...key,
      "session",
      sessionId,
    ]);
    if (saved)
      setDrafts((current) =>
        Object.keys(current).reduce<Record<string, string>>(
          (result, name) =>
            saved.answers.some(
              (a) =>
                `${saved.id}:${a.question_id}` === name &&
                a.text === current[name],
            )
              ? result
              : { ...result, [name]: current[name] },
          {},
        ),
      );
  };
  const mutation = useMutation({
    mutationFn: async (task: () => Promise<void>) => {
      setRequestError("");
      setNotice("");
      await task();
    },
    onError: async (error) => {
      setRequestError(errorCode(error));
      await refresh();
    },
  });
  const run = (task: () => Promise<void>) => mutation.mutate(task);
  const operationForAnswer = (id: string) =>
    operations.find((op) => op.answer_id === id);
  const summaryOperation = operations.find(
    (op) => op.kind === "summary" && op.session_id === sessionId,
  );
  const canGenerate =
    !!resume &&
    !!job.description?.trim() &&
    !isJobDirty &&
    !isJobSaving &&
    aiAvailable &&
    !mutation.isPending;
  const generate = () =>
    run(async () => {
      const identity = `generate:${resume}:${locale}`;
      const op = await service.generate(
        job.id,
        resume,
        locale,
        requestKey(identity),
      );
      requestKeys.current.delete(identity);
      remember(op);
      setLatestOperation(op.id);
      setSelectedVersion("");
      setOffset(0);
      setNotice("");
      if (op.version_id) setSelectedVersion(op.version_id);
      await refresh();
    });
  const feedback = (id: string) =>
    run(async () => {
      const op = await service.feedback(
        job.id,
        sessionId,
        id,
        requestKey(`feedback:${id}`),
      );
      remember(op);
      await refresh();
    });
  const save = (skip: boolean) =>
    run(async () => {
      if (!session || !next) return;
      const body = {
        key: requestKey(`answer:${session.id}:${next.id}:${skip}:${answer}`),
        question_id: next.id,
        text: skip ? "" : answer,
        skip,
        revision: session.revision,
      };
      const saved = await service.answer(job.id, session.id, body);
      cache.setQueryData([...key, "session", session.id], saved);
      setAnswer("");
      setNotice(t("jobs.interview.saved"));
      await refresh();
    });
  const action = (value: "pause" | "resume" | "finish") =>
    run(async () => {
      if (!session) return;
      if (
        value === "finish" &&
        answer.trim() &&
        !window.confirm(t("jobs.interview.unsaved"))
      )
        return;
      const saved = await service.action(
        job.id,
        session.id,
        value,
        session.revision,
        requestKey(`${value}:${session.id}:${session.revision}`),
      );
      cache.setQueryData([...key, "session", session.id], saved);
      setNotice(t("jobs.interview.saved"));
      await refresh();
    });
  const retry = (op: PrepOperation) =>
    run(async () => {
      remember(await service.retry(job.id, op.id));
      await refresh();
    });
  const date = (value: string) => new Date(value).toLocaleString(i18n.language);
  const close = () => {
    if (!answer.trim() || window.confirm(t("jobs.interview.unsaved")))
      onClose();
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) close();
      }}
      className="max-w-4xl"
    >
      <DialogContent
        onClose={close}
        className="ph-no-capture ph-sensitive flex max-h-[90dvh] max-w-none flex-col overflow-hidden p-0"
      >
        <DialogHeader className="shrink-0 border-b p-5 pr-12 text-left">
          <DialogTitle>{t("jobs.interview.title")}</DialogTitle>
          <DialogDescription className="break-words">
            {job.title}
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 space-y-5 overflow-y-auto overscroll-contain p-4 sm:p-6">
          <p className="text-sm text-muted-foreground">
            {t("jobs.interview.assumptions")}
          </p>
          <p className="text-xs text-muted-foreground">
            {t("jobs.interview.quota")}
          </p>
          {(history.isLoading || isLoading) && (
            <p role="status">{t("common.loading")}</p>
          )}
          {history.isError && (
            <div role="alert" className="space-y-2 text-sm">
              <p>{t("jobs.interview.errors.load")}</p>
              <Button variant="outline" onClick={() => void history.refetch()}>
                {t("jobs.interview.retry")}
              </Button>
            </div>
          )}
          {requestError && (
            <p
              role="alert"
              className="rounded-md border border-destructive p-3 text-sm text-destructive"
            >
              {t(`jobs.interview.errors.${requestError}`, {
                defaultValue: t("jobs.interview.errors.generic"),
              })}
            </p>
          )}
          <p role="status" aria-live="polite" className="text-sm">
            {mutation.isPending
              ? t("jobs.interview.saving")
              : latest
                ? t(`jobs.interview.${latest.status}`)
                : notice}
          </p>
          {history.data && !aiAvailable && (
            <p role="status" className="rounded-md border p-3 text-sm">
              {t("jobs.interview.unavailable")}
            </p>
          )}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="interview-resume">
                {t("jobs.interview.source")}
              </Label>
              <Select
                id="interview-resume"
                value={resume}
                onChange={(e) => setResume(e.target.value)}
                disabled={isLoading || mutation.isPending}
              >
                <option value="">{t("jobs.interview.chooseResume")}</option>
                {builderResumes.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.title}
                  </option>
                ))}
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="interview-locale">
                {t("jobs.interview.language")}
              </Label>
              <Select
                id="interview-locale"
                value={locale}
                onChange={(e) => {
                  const value = e.target.value;
                  if (value === "en" || value === "ru" || value === "uk")
                    setLocale(value);
                }}
                disabled={mutation.isPending}
              >
                <option value="en">English</option>
                <option value="ru">Русский</option>
                <option value="uk">Українська</option>
              </Select>
            </div>
          </div>
          {isError && <p role="alert">{t("jobs.interview.errors.resumes")}</p>}
          {!isLoading && !builderResumes.length && (
            <Link
              to="/app/resumes"
              className="inline-flex min-h-11 items-center underline"
            >
              {t("jobs.interview.createResume")}
            </Link>
          )}
          {(!job.description?.trim() || isJobDirty || isJobSaving) && (
            <p className="text-sm text-muted-foreground">
              {t("jobs.interview.saveJob")}
            </p>
          )}
          <Button
            disabled={
              !canGenerate ||
              operations.some((op) => op.kind === "plan" && pending(op))
            }
            onClick={generate}
          >
            {version
              ? t("jobs.interview.regenerate")
              : t("jobs.interview.generate")}
          </Button>
          {operations
            .filter(
              (op) =>
                pending(op) ||
                op.status === "failed" ||
                op.status === "interrupted",
            )
            .map((op) => (
              <div
                key={op.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border p-3 text-sm"
              >
                <p>
                  <strong>{t(`jobs.interview.${op.kind}`)}: </strong>
                  {t(`jobs.interview.${op.status}`)}
                  {op.error_code && (
                    <>
                      {" "}
                      ·{" "}
                      {t(`jobs.interview.errors.${op.error_code}`, {
                        defaultValue: t("jobs.interview.errors.generic"),
                      })}
                    </>
                  )}
                  {!pending(op) && op.attempts >= 3 && (
                    <span className="mt-2 block">
                      {t(
                        op.kind === "plan"
                          ? "jobs.interview.exhaustedPlan"
                          : "jobs.interview.exhaustedReview",
                      )}
                    </span>
                  )}
                </p>
                {!pending(op) &&
                  op.error_code !== "SOURCE_CHANGED" &&
                  op.attempts < 3 && (
                    <Button
                      variant="outline"
                      disabled={!aiAvailable || mutation.isPending}
                      onClick={() => retry(op)}
                    >
                      {t("jobs.interview.retrySafe")}
                    </Button>
                  )}
              </div>
            ))}
          <div className="space-y-2">
            <Label htmlFor="interview-history">
              {t("jobs.interview.history")}
            </Label>
            <Select
              id="interview-history"
              disabled={mutation.isPending}
              value={versionId}
              onChange={(e) => setSelectedVersion(e.target.value)}
            >
              <option value="">{t("jobs.interview.empty")}</option>
              {history.data?.versions.map((v) => (
                <option key={v.id} value={v.id}>
                  {date(v.created_at)} · {v.resume_title} ·{" "}
                  {v.locale.toUpperCase()}
                </option>
              ))}
            </Select>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                disabled={offset === 0}
                onClick={() => {
                  setOffset(Math.max(0, offset - 20));
                  setSelectedVersion("");
                }}
              >
                {t("jobs.interview.newer")}
              </Button>
              <Button
                variant="ghost"
                disabled={
                  (history.data?.versions.length ?? 0) < 20 &&
                  (history.data?.sessions.length ?? 0) < 20 &&
                  (history.data?.operations.length ?? 0) < 30
                }
                onClick={() => {
                  setOffset(offset + 20);
                  setSelectedVersion("");
                }}
              >
                {t("jobs.interview.older")}
              </Button>
            </div>
          </div>
          <div
            className="flex flex-wrap gap-2"
            role="group"
            aria-label={t("jobs.interview.view")}
          >
            <Button
              variant={tab === "plan" ? "default" : "outline"}
              aria-pressed={tab === "plan"}
              onClick={() => setTab("plan")}
            >
              {t("jobs.interview.plan")}
            </Button>
            <Button
              variant={tab === "practice" ? "default" : "outline"}
              aria-pressed={tab === "practice"}
              onClick={() => setTab("practice")}
            >
              {t("jobs.interview.practice")}
            </Button>
          </div>
          {versionQuery.isLoading && <p role="status">{t("common.loading")}</p>}
          {versionQuery.isError && (
            <p role="alert">{t("jobs.interview.errors.load")}</p>
          )}
          {version?.stale && (
            <p
              role="status"
              className="rounded-md border border-amber-500 p-3 text-sm"
            >
              {t("jobs.interview.stale")}
            </p>
          )}
          {tab === "plan" && version && (
            <>
              <p className="text-xs text-muted-foreground">
                {version.resume_title} · {date(version.created_at)} ·{" "}
                {version.locale.toUpperCase()}
              </p>
              <PlanContent
                version={version}
                busy={mutation.isPending}
                onStudied={(id, value) =>
                  run(async () => {
                    await service.studied(job.id, version.id, id, value);
                    await refresh();
                  })
                }
              />
            </>
          )}
          {tab === "practice" && (
            <div className="space-y-5">
              <Button
                disabled={!version || mutation.isPending}
                onClick={() =>
                  run(async () => {
                    if (!version) return;
                    if (
                      answer.trim() &&
                      !window.confirm(t("jobs.interview.unsaved"))
                    )
                      return;
                    const started = await service.start(
                      job.id,
                      version.id,
                      requestKey(`start:${version.id}`),
                    );
                    requestKeys.current.delete(`start:${version.id}`);
                    setSelectedSession(started.id);
                    setOffset(0);
                    setAnswer("");
                    await refresh();
                  })
                }
              >
                {t("jobs.interview.start")}
              </Button>
              <div className="space-y-2">
                <Label htmlFor="interview-sessions">
                  {t("jobs.interview.sessions")}
                </Label>
                <Select
                  id="interview-sessions"
                  disabled={mutation.isPending}
                  value={sessionId}
                  onChange={(e) => {
                    if (
                      answer.trim() &&
                      !window.confirm(t("jobs.interview.unsaved"))
                    )
                      return;
                    setSelectedSession(e.target.value);
                    setAnswer("");
                  }}
                >
                  <option value="">{t("jobs.interview.chooseSession")}</option>
                  {history.data?.sessions.map((s) => (
                    <option key={s.id} value={s.id}>
                      {date(s.created_at)} · {t(`jobs.interview.${s.status}`)}
                    </option>
                  ))}
                </Select>
              </div>
              {sessionQuery.isLoading && (
                <p role="status">{t("common.loading")}</p>
              )}
              {sessionQuery.isError && (
                <p role="alert">{t("jobs.interview.errors.load")}</p>
              )}
              {session && (
                <>
                  <p className="text-sm">
                    {t(`jobs.interview.${session.status}`)} ·{" "}
                    {session.answers.length}/{session.questions.length} ·{" "}
                    {t("jobs.interview.savedServer")}
                  </p>
                  {practiceStale && (
                    <p
                      role="status"
                      className="rounded-md border border-amber-500 p-3 text-sm"
                    >
                      {t("jobs.interview.stale")}
                    </p>
                  )}
                  <div className="flex flex-wrap gap-2">
                    {session.status === "active" && (
                      <Button
                        variant="outline"
                        disabled={mutation.isPending}
                        onClick={() => action("pause")}
                      >
                        {t("jobs.interview.pause")}
                      </Button>
                    )}
                    {session.status === "paused" && (
                      <Button
                        disabled={mutation.isPending}
                        onClick={() => action("resume")}
                      >
                        {t("jobs.interview.resume")}
                      </Button>
                    )}
                    {session.status !== "completed" && (
                      <Button
                        variant="outline"
                        disabled={mutation.isPending}
                        onClick={() => action("finish")}
                      >
                        {t("jobs.interview.finish")}
                      </Button>
                    )}
                  </div>
                  {next && session.status === "active" && (
                    <div className="space-y-4 rounded-lg border p-4">
                      <div
                        ref={questionHeading}
                        tabIndex={-1}
                        className="outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      >
                        <QuestionContent question={next} />
                      </div>
                      <Label htmlFor="interview-answer">
                        {t("jobs.interview.answer")}
                      </Label>
                      <Textarea
                        id="interview-answer"
                        value={answer}
                        onChange={(e) => {
                          if (interviewTextLength(e.target.value) <= 12000) {
                            setAnswer(e.target.value);
                            setRequestError((code) =>
                              code === "ANSWER_TOO_LONG" ? "" : code,
                            );
                          } else {
                            setRequestError("ANSWER_TOO_LONG");
                          }
                        }}
                        aria-describedby="interview-answer-count"
                        className="min-h-40"
                        disabled={mutation.isPending}
                      />
                      <p
                        id="interview-answer-count"
                        role="status"
                        aria-live="polite"
                        className="text-xs text-muted-foreground"
                      >
                        {interviewTextLength(answer)}/12000 ·{" "}
                        {t("jobs.interview.draft")}
                        <span className="block">
                          {t("jobs.interview.capacity", {
                            bytes: session.answer_bytes_remaining,
                          })}
                        </span>
                      </p>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          disabled={!answer.trim() || mutation.isPending}
                          onClick={() => save(false)}
                        >
                          {t("jobs.interview.saveAnswer")}
                        </Button>
                        <Button
                          variant="outline"
                          disabled={mutation.isPending}
                          onClick={() => save(true)}
                        >
                          {t("jobs.interview.skip")}
                        </Button>
                      </div>
                    </div>
                  )}
                  {!next && session.status !== "completed" && (
                    <p role="status">{t("jobs.interview.allAnswered")}</p>
                  )}
                  {session.answers.map((a) => {
                    const op = operationForAnswer(a.id);
                    return (
                      <article
                        key={a.id}
                        className="space-y-3 rounded-lg border p-4"
                      >
                        <h4 className="font-medium">
                          {
                            session.questions.find(
                              (q) => q.id === a.question_id,
                            )?.question
                          }
                        </h4>
                        <p className="whitespace-pre-wrap break-words text-sm">
                          {a.skipped ? t("jobs.interview.skipped") : a.text}
                        </p>
                        {a.feedback ? (
                          <ReviewContent review={a.feedback} />
                        ) : (
                          !a.skipped && (
                            <>
                              <p className="text-xs text-muted-foreground">
                                {t("jobs.interview.savedFeedbackPending")}
                              </p>
                              {!op && (
                                <Button
                                  variant="outline"
                                  disabled={
                                    !aiAvailable ||
                                    !!practiceStale ||
                                    mutation.isPending
                                  }
                                  onClick={() => feedback(a.id)}
                                >
                                  {t("jobs.interview.getFeedback")}
                                </Button>
                              )}
                            </>
                          )
                        )}
                      </article>
                    );
                  })}
                  {session.status === "completed" && (
                    <div className="space-y-3">
                      <h3 className="font-semibold">
                        {t("jobs.interview.summary")}
                      </h3>
                      {session.summary ? (
                        <ReviewContent review={session.summary} />
                      ) : (
                        <>
                          <p className="text-sm">
                            {t("jobs.interview.summaryPending")}
                          </p>
                          {!summaryOperation && (
                            <Button
                              disabled={
                                !aiAvailable ||
                                !!practiceStale ||
                                mutation.isPending
                              }
                              onClick={() =>
                                run(async () => {
                                  const op = await service.summary(
                                    job.id,
                                    session.id,
                                    requestKey(`summary:${session.id}`),
                                  );
                                  remember(op);
                                  await refresh();
                                })
                              }
                            >
                              {t("jobs.interview.getSummary")}
                            </Button>
                          )}
                        </>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
