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
import type { JobDTO, MatchScoreResponse } from "@/shared/types/api";

export interface EditableFields {
  title: string;
  company_id: string;
  url: string;
  source: string;
  description: string;
  notes: string;
}

export function fieldsFromJob(job: JobDTO): EditableFields {
  return {
    title: job.title,
    company_id: job.company_id ?? "",
    url: job.url ?? "",
    source: job.source ?? "",
    description: job.description ?? "",
    notes: job.notes ?? "",
  };
}

export function hasChanges(fields: EditableFields, job: JobDTO): boolean {
  return (
    fields.title !== job.title ||
    fields.company_id !== (job.company_id ?? "") ||
    fields.url !== (job.url ?? "") ||
    fields.source !== (job.source ?? "") ||
    fields.description !== (job.description ?? "") ||
    fields.notes !== (job.notes ?? "")
  );
}

export function resumeSelectValue(job: JobDTO): string {
  if (!job.resume) return "";
  return `${job.resume.type}:${job.resume.id}`;
}

interface UseJobDetailMutationsParams {
  readonly id: string | undefined;
  readonly effectiveMatchResumeId: string;
  readonly setFields: (fields: EditableFields) => void;
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

  const updateMutation = useMutation({
    mutationFn: (data: Partial<EditableFields>) =>
      jobsService.update(id!, {
        title: data.title,
        company_id: data.company_id || undefined,
        url: data.url || undefined,
        source: data.source || undefined,
        description: data.description || undefined,
        notes: data.notes || undefined,
      }),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      setFields(fieldsFromJob(updated));
      showSuccessNotification(t("jobs.updateSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.updateError"));
    },
  });

  const changeResumeMutation = useMutation({
    mutationFn: (value: string) => {
      if (value.startsWith("uploaded:")) {
        return jobsService.update(id!, {
          resume_id: value.slice("uploaded:".length),
        });
      }
      if (value.startsWith("builder:")) {
        return jobsService.update(id!, {
          resume_builder_id: value.slice("builder:".length),
        });
      }
      // Empty string clears the attached resume server-side
      return jobsService.update(id!, { resume_id: "" });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.updateSuccess"));
    },
    onError: (err: Error) => {
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

  const archiveMutation = useMutation({
    mutationFn: jobsService.archive,
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

  const checkMatchMutation = useMutation({
    mutationFn: () => {
      if (!effectiveMatchResumeId) {
        return Promise.reject(new Error(t("jobs.matchScore.noResume")));
      }
      return matchScoreService.checkMatch(id!, effectiveMatchResumeId);
    },
    onSuccess: (data) => {
      setMatchScore(data);
      setMatchScoreError(null);
      setIsPricingModalOpen(false);
    },
    onError: (err: Error) => {
      if (err instanceof ApiError) {
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
    archiveMutation,
    deleteMutation,
    completeCurrentStageMutation,
    addCommentMutation,
    checkMatchMutation,
  };
}
