import { useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import { resumesService } from "@/services/resumesService";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { commentsService } from "@/services/commentsService";
import { matchScoreService } from "@/services/matchScoreService";
import { companiesService } from "@/services/companiesService";
import { ApiError } from "@/services/api";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { SkeletonDetail } from "@/shared/ui/Skeleton";
import { ErrorState } from "@/shared/ui/ErrorState";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { MatchScoreCard } from "@/features/jobs/components/MatchScoreCard";
import { Timeline } from "@/features/jobs/components/Timeline";
import { AddStageModal } from "@/features/jobs/modals/AddStageModal";
import { UpdateJobStatusModal } from "@/features/jobs/modals/UpdateJobStatusModal";
import { CompanySelectWithQuickAdd } from "@/features/jobs/components/CompanySelectWithQuickAdd";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import {
  ArrowLeft,
  Calendar,
  ExternalLink,
  Save,
  Archive,
  Sparkles,
  Loader2,
  Heart,
  Plus,
  Edit,
  MessageSquarePlus,
  Trash2,
} from "lucide-react";
import { formatDistanceToNow, format } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type { JobDTO, MatchScoreResponse } from "@/shared/types/api";

interface EditableFields {
  title: string;
  company_id: string;
  url: string;
  source: string;
  description: string;
  notes: string;
}

function fieldsFromJob(job: JobDTO): EditableFields {
  return {
    title: job.title,
    company_id: job.company_id ?? "",
    url: job.url ?? "",
    source: job.source ?? "",
    description: job.description ?? "",
    notes: job.notes ?? "",
  };
}

function hasChanges(fields: EditableFields, job: JobDTO): boolean {
  return (
    fields.title !== job.title ||
    fields.company_id !== (job.company_id ?? "") ||
    fields.url !== (job.url ?? "") ||
    fields.source !== (job.source ?? "") ||
    fields.description !== (job.description ?? "") ||
    fields.notes !== (job.notes ?? "")
  );
}

function resumeSelectValue(job: JobDTO): string {
  if (!job.resume) return "";
  return `${job.resume.type}:${job.resume.id}`;
}

export default function JobDetail() {
  usePageMeta({ titleKey: "jobs.details", noindex: true });
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [fields, setFields] = useState<EditableFields | null>(null);
  const [isAddStageModalOpen, setIsAddStageModalOpen] = useState(false);
  const [isUpdateStatusModalOpen, setIsUpdateStatusModalOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [newComment, setNewComment] = useState("");
  const [selectedMatchResumeId, setSelectedMatchResumeId] = useState<
    string | null
  >(null);
  const [matchScore, setMatchScore] = useState<MatchScoreResponse | null>(null);
  const [matchScoreError, setMatchScoreError] = useState<string | null>(null);
  const [isPricingModalOpen, setIsPricingModalOpen] = useState(false);

  const {
    data: job,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["job", id],
    queryFn: () => jobsService.getById(id!),
    enabled: !!id,
  });

  const { data: stages } = useQuery({
    queryKey: ["job-stages", id],
    queryFn: () => jobsService.listStages(id!),
    enabled: !!id,
  });

  const { data: companiesData } = useQuery({
    queryKey: ["companies"],
    queryFn: () => companiesService.list({ limit: 100, offset: 0 }),
    enabled: !!id,
  });

  const { data: resumesData } = useQuery({
    queryKey: ["resumes"],
    queryFn: () => resumesService.list({ limit: 100, offset: 0 }),
    enabled: !!id,
  });

  const { data: builderResumes } = useQuery({
    queryKey: ["resume-builder"],
    queryFn: () => resumeBuilderService.list(),
    enabled: !!id,
  });

  // Initialize fields from job data once loaded
  const editableFields = useMemo(() => {
    if (!job) return null;
    if (fields) return fields;
    return fieldsFromJob(job);
  }, [job, fields]);

  const isDirty =
    job && editableFields ? hasChanges(editableFields, job) : false;

  const updateField = (key: keyof EditableFields, value: string) => {
    setFields((prev) => {
      const base = prev ?? (job ? fieldsFromJob(job) : null);
      if (!base) return prev;
      return { ...base, [key]: value };
    });
  };

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

  // Default match-score resume: attached uploaded resume, else explicit selection
  const effectiveMatchResumeId =
    selectedMatchResumeId ??
    (job?.resume?.type === "uploaded" ? job.resume.id : "");

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

  const handleSave = () => {
    if (!editableFields || !editableFields.title.trim()) return;
    updateMutation.mutate(editableFields);
  };

  const handleDiscard = () => {
    if (job) setFields(fieldsFromJob(job));
  };

  const handleAddComment = (e: React.FormEvent) => {
    e.preventDefault();
    if (newComment.trim()) {
      addCommentMutation.mutate(newComment.trim());
    }
  };

  const isValidUrl = (url: string) => {
    try {
      const protocol = new URL(url).protocol;
      return protocol === "http:" || protocol === "https:";
    } catch {
      return false;
    }
  };

  const uploadedResumes = resumesData?.items ?? [];
  const builderResumesList = builderResumes ?? [];
  const jobComments = job?.job_comments || [];

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate("/app/jobs")}>
          <ArrowLeft className="h-4 w-4" />
          {t("jobs.backToJobs")}
        </Button>
        <SkeletonDetail />
      </div>
    );
  }

  if (isError || !job || !editableFields) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate("/app/jobs")}>
          <ArrowLeft className="h-4 w-4" />
          {t("jobs.backToJobs")}
        </Button>
        <ErrorState
          message={error?.message || t("jobs.notFound")}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const notesDirty = editableFields.notes !== (job.notes ?? "");

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Button variant="ghost" onClick={() => navigate("/app/jobs")}>
          <ArrowLeft className="h-4 w-4" />
          {t("jobs.backToJobs")}
        </Button>
        <div className="flex flex-wrap gap-2">
          {isDirty && (
            <>
              <Button variant="outline" size="sm" onClick={handleDiscard}>
                {t("common.cancel")}
              </Button>
              <Button
                size="sm"
                onClick={handleSave}
                disabled={
                  updateMutation.isPending || !editableFields.title.trim()
                }
              >
                {updateMutation.isPending ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Save className="h-4 w-4 mr-2" />
                )}
                {t("common.save")}
              </Button>
            </>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => toggleFavoriteMutation.mutate()}
            disabled={toggleFavoriteMutation.isPending}
            aria-label={
              job.is_favorite
                ? t("common.removeFromFavorites")
                : t("common.addToFavorites")
            }
          >
            <Heart
              className={`h-4 w-4 ${job.is_favorite ? "fill-red-500 text-red-500" : ""}`}
            />
          </Button>
          {job.status !== "archived" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => archiveMutation.mutate(job.id)}
              disabled={archiveMutation.isPending}
              aria-label={t("jobs.archive")}
            >
              <Archive className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">{t("jobs.archive")}</span>
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => setIsDeleteConfirmOpen(true)}
            disabled={deleteMutation.isPending}
            aria-label={t("common.delete")}
          >
            <Trash2 className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">{t("common.delete")}</span>
          </Button>
        </div>
      </div>

      {/* Main info */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-4">
            <Input
              value={editableFields.title}
              onChange={(e) => updateField("title", e.target.value)}
              className="text-2xl font-bold border-transparent hover:border-input focus:border-input bg-transparent h-auto py-1 px-2 -ml-2"
              placeholder={t("jobs.titlePlaceholder")}
            />
            <StatusBadge status={job.status} />
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <CompanySelectWithQuickAdd
            companies={companiesData?.items ?? []}
            value={editableFields.company_id}
            onChange={(val) => updateField("company_id", val)}
          />

          <div className="space-y-2">
            <Label htmlFor="source">{t("jobs.source")}</Label>
            <Input
              id="source"
              value={editableFields.source}
              onChange={(e) => updateField("source", e.target.value)}
              placeholder={t("jobs.sourcePlaceholder")}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="url">{t("jobs.url")}</Label>
            <div className="flex items-center gap-2">
              <Input
                id="url"
                type="url"
                value={editableFields.url}
                onChange={(e) => updateField("url", e.target.value)}
                placeholder="https://example.com/jobs/123"
                className="flex-1"
              />
              {editableFields.url && isValidUrl(editableFields.url) && (
                <a
                  href={editableFields.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="shrink-0 text-primary hover:text-primary/80"
                >
                  <ExternalLink className="h-4 w-4" />
                </a>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Calendar className="h-4 w-4" />
            <span>
              {t("jobs.createdDate")}{" "}
              {formatDistanceToNow(new Date(job.created_at), {
                addSuffix: true,
                locale: dateLocale,
              })}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Pipeline */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t("jobs.pipelineTitle")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm text-muted-foreground">
                {t("jobs.status")}
              </p>
              <StatusBadge status={job.status} />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsUpdateStatusModalOpen(true)}
            >
              <Edit className="h-4 w-4 mr-2" />
              {t("jobs.changeStatus")}
            </Button>
          </div>

          {job.applied_at && (
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">
                {t("jobs.applied")}{" "}
                {format(new Date(job.applied_at), "PP", {
                  locale: dateLocale,
                })}{" "}
                (
                {formatDistanceToNow(new Date(job.applied_at), {
                  addSuffix: true,
                  locale: dateLocale,
                })}
                )
              </span>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="attached-resume">{t("jobs.resumeLabel")}</Label>
            <select
              id="attached-resume"
              value={resumeSelectValue(job)}
              onChange={(e) => changeResumeMutation.mutate(e.target.value)}
              disabled={changeResumeMutation.isPending}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="">{t("jobs.selectResume")}</option>
              {uploadedResumes.length > 0 && (
                <optgroup label={t("jobs.uploadedResumes")}>
                  {uploadedResumes.map((resume) => (
                    <option
                      key={`uploaded:${resume.id}`}
                      value={`uploaded:${resume.id}`}
                    >
                      {resume.title}
                    </option>
                  ))}
                </optgroup>
              )}
              {builderResumesList.length > 0 && (
                <optgroup label={t("jobs.builderResumes")}>
                  {builderResumesList.map((rb) => (
                    <option key={`builder:${rb.id}`} value={`builder:${rb.id}`}>
                      {rb.title}
                    </option>
                  ))}
                </optgroup>
              )}
            </select>
          </div>
        </CardContent>
      </Card>

      {/* Description */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t("jobs.description")}</CardTitle>
        </CardHeader>
        <CardContent>
          <Textarea
            value={editableFields.description}
            onChange={(e) => updateField("description", e.target.value)}
            placeholder={t("jobs.descriptionPlaceholder")}
            className="min-h-[120px]"
          />
        </CardContent>
      </Card>

      {/* Notes */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t("jobs.notes")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Textarea
            value={editableFields.notes}
            onChange={(e) => updateField("notes", e.target.value)}
            placeholder={t("jobs.notesPlaceholder")}
            className="min-h-[80px]"
          />
          <div className="flex justify-end">
            <Button
              size="sm"
              onClick={handleSave}
              disabled={
                !notesDirty ||
                updateMutation.isPending ||
                !editableFields.title.trim()
              }
            >
              {updateMutation.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Save className="h-4 w-4 mr-2" />
              )}
              {t("common.save")}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Check Match */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            {t("jobs.matchScore.checkMatch")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="resume-select">{t("jobs.selectResume")}</Label>
              <select
                id="resume-select"
                value={effectiveMatchResumeId}
                onChange={(e) => setSelectedMatchResumeId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">{t("jobs.selectResume")}</option>
                {uploadedResumes.map((resume) => (
                  <option key={resume.id} value={resume.id}>
                    {resume.title}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={() => checkMatchMutation.mutate()}
              disabled={!effectiveMatchResumeId || checkMatchMutation.isPending}
            >
              {checkMatchMutation.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Sparkles className="h-4 w-4 mr-2" />
              )}
              {checkMatchMutation.isPending
                ? t("jobs.matchScore.checking")
                : t("jobs.matchScore.checkMatch")}
            </Button>
          </div>

          {matchScoreError && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
              {matchScoreError}
            </div>
          )}
        </CardContent>
      </Card>

      {matchScore && <MatchScoreCard data={matchScore} />}

      {/* Comments */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">{t("jobs.comments")}</CardTitle>
        </CardHeader>
        <CardContent>
          {jobComments.length > 0 && (
            <div className="space-y-3 mb-4">
              {jobComments.map((comment) => (
                <div
                  key={comment.id}
                  className="rounded-lg border bg-muted/50 p-3"
                >
                  <p className="text-sm whitespace-pre-wrap">
                    {comment.content}
                  </p>
                  <p className="text-xs text-muted-foreground mt-2">
                    {formatDistanceToNow(new Date(comment.created_at), {
                      addSuffix: true,
                      locale: dateLocale,
                    })}
                  </p>
                </div>
              ))}
            </div>
          )}

          <form onSubmit={handleAddComment} className="space-y-2">
            <Textarea
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder={t("jobs.commentPlaceholder")}
              className="flex-1"
              rows={3}
            />
            <Button
              type="submit"
              size="sm"
              disabled={!newComment.trim() || addCommentMutation.isPending}
            >
              <MessageSquarePlus className="h-4 w-4 mr-2" />
              {t("jobs.addComment")}
            </Button>
          </form>
        </CardContent>
      </Card>

      {/* Timeline */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle>{t("jobs.timeline")}</CardTitle>
          <Button size="sm" onClick={() => setIsAddStageModalOpen(true)}>
            <Plus className="h-4 w-4" />
            {t("jobs.addNewStage")}
          </Button>
        </CardHeader>
        <CardContent>
          <Timeline
            stages={stages || []}
            jobId={id!}
            stageComments={job.stage_comments || []}
          />
        </CardContent>
      </Card>

      <AddStageModal
        open={isAddStageModalOpen}
        onOpenChange={setIsAddStageModalOpen}
        jobId={id!}
      />

      <UpdateJobStatusModal
        key={`${job.status}-${isUpdateStatusModalOpen}`}
        open={isUpdateStatusModalOpen}
        onOpenChange={setIsUpdateStatusModalOpen}
        jobId={id!}
        currentStatus={job.status}
      />

      {/* Delete confirmation */}
      <Dialog open={isDeleteConfirmOpen} onOpenChange={setIsDeleteConfirmOpen}>
        <DialogContent onClose={() => setIsDeleteConfirmOpen(false)}>
          <DialogHeader>
            <DialogTitle>{t("jobs.delete")}</DialogTitle>
            <DialogDescription>{t("jobs.deleteConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-6">
            <Button
              variant="outline"
              onClick={() => setIsDeleteConfirmOpen(false)}
              disabled={deleteMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => deleteMutation.mutate()}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t("common.loading")}
                </>
              ) : (
                t("common.delete")
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <PricingModal
        open={isPricingModalOpen}
        onOpenChange={setIsPricingModalOpen}
      />
    </div>
  );
}
