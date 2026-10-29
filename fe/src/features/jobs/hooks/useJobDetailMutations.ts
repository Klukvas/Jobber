import { useAIConsentSignal } from "@/shared/hooks/useAIConsentSignal";
import { isAIActionCancelled } from "@/shared/lib/aiCredits";
import { useEffect, useLayoutEffect, useRef } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { jobsService } from "@/services/jobsService";
import { commentsService } from "@/services/commentsService";
import { matchScoreService } from "@/services/matchScoreService";
import { ApiError } from "@/services/api";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type {
  JobDTO,
  MatchScoreResponse,
  UpdateJobRequest,
} from "@/shared/types/api";

export interface EditableFields {
  title: string;
  company_id: string;
  url: string;
  source: string;
  description: string;
}

export function fieldsFromJob(job: JobDTO): EditableFields {
  return {
    title: job.title,
    company_id: job.company_id ?? "",
    url: job.url ?? "",
    source: job.source ?? "",
    description: job.description ?? "",
  };
}

export function hasChanges(fields: EditableFields, job: JobDTO): boolean {
  return (
    fields.title !== job.title ||
    fields.company_id !== (job.company_id ?? "") ||
    fields.url !== (job.url ?? "") ||
    fields.source !== (job.source ?? "") ||
    fields.description !== (job.description ?? "")
  );
}

/**
 * The PATCH body for an edited job.
 *
 * Only the fields the caller actually holds are sent, and they are sent
 * verbatim: the API distinguishes an absent field (leave it alone) from an
 * empty string (clear it), so mapping `""` to `undefined` here silently threw
 * away every attempt to erase a company, URL, source, or description.
 */
export function jobUpdatePayload(
  fields: Partial<EditableFields>,
): UpdateJobRequest {
  const payload: UpdateJobRequest = {};
  if (fields.title !== undefined) payload.title = fields.title.trim();
  if (fields.company_id !== undefined) payload.company_id = fields.company_id;
  if (fields.url !== undefined) payload.url = fields.url;
  if (fields.source !== undefined) payload.source = fields.source;
  if (fields.description !== undefined)
    payload.description = fields.description;
  return payload;
}

export function resumeSelectValue(job: JobDTO): string {
  if (!job.resume) return "";
  return `${job.resume.type}:${job.resume.id}`;
}

interface UseJobDetailMutationsParams {
  readonly id: string | undefined;
  readonly effectiveMatchResumeId: string;
  readonly setFields: React.Dispatch<React.SetStateAction<EditableFields | null>>;
  readonly setNewComment: (value: string) => void;
  readonly setMatchScore: (data: MatchScoreResponse | null) => void;
  readonly setMatchScoreError: (message: string | null) => void;
  readonly setIsPricingModalOpen: (open: boolean) => void;
}

export function useJobDetailMutations({
  id,
  effectiveMatchResumeId,
  setFields,
  setNewComment,
  setMatchScore,
  setMatchScoreError,
  setIsPricingModalOpen,
}: UseJobDetailMutationsParams) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => { active.current = false; };
  }, []);


  const updateMutation = useMutation({
    mutationFn: async (data: Partial<EditableFields>) => {
      const jobId = id;
      if (!jobId || (data.title !== undefined && !data.title.trim())) throw new Error(t("jobs.titleRequired"));
      const updated = await jobsService.update(jobId, jobUpdatePayload(data));
      return { updated, jobId, submitted: data };
    },
    onSuccess: ({ updated, jobId, submitted }) => {
      queryClient.invalidateQueries({ queryKey: ["job", jobId] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      if (!active.current || updated.id !== id) return;
      setFields((current) => {
        // A response acknowledges its snapshot; typing during that save belongs
        // to the next one and must not be replaced by the older response.
        const editedSinceSubmit = current && Object.entries(submitted).some(([key, value]) => {
          const field = key as keyof EditableFields;
          return value !== undefined && (field === "title" ? current.title.trim() !== value.trim() : current[field] !== value);
        });
        return editedSinceSubmit ? current : fieldsFromJob(updated);
      });
      showSuccessNotification(t("jobs.updateSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.updateError"));
    },
  });

  const changeResumeMutation = useMutation({
    mutationFn: async (value: string) => {
      const jobId = id;
      if (!jobId) throw new Error(t("jobs.updateError"));
      if (value.startsWith("uploaded:")) {
        return { jobId, updated: await jobsService.update(jobId, {
          resume_id: value.slice("uploaded:".length), resume_builder_id: "",
        }) };
      }
      if (value.startsWith("builder:")) {
        return { jobId, updated: await jobsService.update(jobId, {
          resume_builder_id: value.slice("builder:".length), resume_id: "",
        }) };
      }
      // Empty string clears the attached resume server-side
      return { jobId, updated: await jobsService.update(jobId, { resume_id: "", resume_builder_id: "" }) };
    },
    onSuccess: ({ jobId, updated }) => {
      queryClient.setQueryData(["job", jobId], updated);
      queryClient.invalidateQueries({ queryKey: ["job", jobId] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      if (!active.current) return;
      showSuccessNotification(t("jobs.updateSuccess"));
    },
    onError: (err: Error) => {
      if (!active.current) return;
      showErrorNotification(err.message || t("jobs.updateError"));
    },
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: () => jobsService.toggleFavorite(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
    onError: () => {
      showErrorNotification(t("common.error"));
    },
  });

  // Move the card to a pipeline column (the single write path for its state).
  const moveMutation = useMutation({
    mutationFn: (stageTemplateId: string) =>
      jobsService.move(id!, { stage_template_id: stageTemplateId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.board.moveSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.board.moveError"));
    },
  });

  const archiveMutation = useMutation({
    mutationFn: () => jobsService.archive(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      showSuccessNotification(t("jobs.archiveSuccess"));
      navigate("/app/jobs");
    },
    onError: () => {
      showErrorNotification(t("jobs.archiveError"));
    },
  });

  const unarchiveMutation = useMutation({
    mutationFn: () => jobsService.unarchive(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      showSuccessNotification(t("jobs.unarchiveSuccess"));
    },
    onError: () => {
      showErrorNotification(t("jobs.unarchiveError"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => jobsService.delete(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.deleteSuccess"));
      navigate("/app/jobs");
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.deleteError"));
    },
  });

  // Completes the current stage straight from the Pipeline block (the same
  // action as the Complete button on that stage down in the Timeline)
  const completeCurrentStageMutation = useMutation({
    mutationFn: (stageId: string) =>
      jobsService.updateStage(id!, stageId, {
        status: "completed",
        completed_at: new Date().toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      queryClient.invalidateQueries({ queryKey: ["job-stages", id] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.stageCompletedSuccess"));
    },
    onError: () => {
      showErrorNotification(t("jobs.stageStatusUpdateError"));
    },
  });

  const addCommentMutation = useMutation({
    mutationFn: (content: string) =>
      commentsService.create({
        job_id: id!,
        content,
      }),
    onSuccess: () => {
      // Invalidate job query to refresh embedded comments
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      setNewComment("");
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.commentAddedError"));
    },
  });

  const updateCommentMutation = useMutation({
    mutationFn: ({
      commentId,
      content,
    }: {
      commentId: string;
      content: string;
    }) => commentsService.update(commentId, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      showSuccessNotification(t("jobs.commentUpdated"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.commentUpdateError"));
    },
  });

  const deleteCommentMutation = useMutation({
    mutationFn: (commentId: string) => commentsService.delete(commentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      showSuccessNotification(t("jobs.commentDeleted"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.commentDeleteError"));
    },
  });

  const getConsentSignal = useAIConsentSignal();
  const matchContext = useRef(new AbortController());
  useLayoutEffect(() => {
    const controller = new AbortController();
    matchContext.current = controller;
    return () => controller.abort();
  }, [id, effectiveMatchResumeId]);
  const checkMatchMutation = useMutation({
    mutationFn: async () => {
      if (!id || !effectiveMatchResumeId) {
        return Promise.reject(new Error(t("jobs.matchScore.noResume")));
      }
      const consentSignal = AbortSignal.any([getConsentSignal(), matchContext.current.signal]);
      try {
        const data = await matchScoreService.checkMatch(id, effectiveMatchResumeId, { consentSignal });
        consentSignal.throwIfAborted();
        return data;
      } catch (error) {
        consentSignal.throwIfAborted();
        throw error;
      }
    },
    onSuccess: (data) => {
      setMatchScore(data);
      setMatchScoreError(null);
      setIsPricingModalOpen(false);
    },
    onError: (err: Error) => {
      if (isAIActionCancelled(err)) return;
      if (err instanceof ApiError) {
        if (["CREDIT_UNAVAILABLE", "AI_BUDGET_REACHED", "AI_OPERATION_CONFLICT"].includes(err.code)) {
          setMatchScoreError(err.message);
          return;
        }
        if (err.code === "PLAN_LIMIT_REACHED") {
          setIsPricingModalOpen(true);
          queryClient.invalidateQueries({ queryKey: ["subscription"] });
        } else if (err.code === "JOB_DESCRIPTION_EMPTY") {
          setMatchScoreError(t("jobs.matchScore.noDescription"));
        } else if (err.code === "RESUME_FILE_EMPTY") {
          setMatchScoreError(t("jobs.matchScore.noResumeFile"));
        } else if (err.code === "AI_NOT_CONFIGURED") {
          setMatchScoreError(t("jobs.matchScore.aiNotAvailable"));
        } else {
          setMatchScoreError(t("jobs.matchScore.error"));
        }
      } else {
        setMatchScoreError(err.message || t("jobs.matchScore.error"));
      }
    },
  });

  return {
    updateMutation,
    changeResumeMutation,
    toggleFavoriteMutation,
    moveMutation,
    archiveMutation,
    unarchiveMutation,
    deleteMutation,
    completeCurrentStageMutation,
    addCommentMutation,
    updateCommentMutation,
    deleteCommentMutation,
    checkMatchMutation,
  };
}
