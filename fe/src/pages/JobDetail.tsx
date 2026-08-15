import { useState, useMemo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import { resumesService } from "@/services/resumesService";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { companiesService } from "@/services/companiesService";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Textarea } from "@/shared/ui/Textarea";
import { SkeletonDetail } from "@/shared/ui/Skeleton";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { Timeline } from "@/features/jobs/components/Timeline";
import { JobReminders } from "@/features/jobs/components/JobReminders";
import { JobTags } from "@/features/jobs/components/JobTags";
import { JobDetailHeader } from "@/features/jobs/components/JobDetailHeader";
import { JobInfoCard } from "@/features/jobs/components/JobInfoCard";
import { JobPipelineCard } from "@/features/jobs/components/JobPipelineCard";
import { JobMatchScoreCard } from "@/features/jobs/components/JobMatchScoreCard";
import { JobCommentsSection } from "@/features/jobs/components/JobCommentsSection";
import { AddStageModal } from "@/features/jobs/modals/AddStageModal";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import { ArrowLeft, Save, Loader2, Plus } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { stageTemplatesService } from "@/services/stageTemplatesService";
import {
  fieldsFromJob,
  hasChanges,
  useJobDetailMutations,
  type EditableFields,
} from "@/features/jobs/hooks/useJobDetailMutations";
import type { MatchScoreResponse } from "@/shared/types/api";

export default function JobDetail() {
  usePageMeta({ titleKey: "jobs.details", noindex: true });
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [fields, setFields] = useState<EditableFields | null>(null);
  const [isAddStageModalOpen, setIsAddStageModalOpen] = useState(false);
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

  // The user's ordered stage templates ARE the pipeline columns; the selector
  // moves the card between them.
  const { data: templatesData } = useQuery({
    queryKey: ["stage-templates"],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
    staleTime: 5 * 60 * 1000,
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

  // Default match-score resume: attached uploaded resume, else explicit selection
  const effectiveMatchResumeId =
    selectedMatchResumeId ??
    (job?.resume?.type === "uploaded" ? job.resume.id : "");

  const {
    updateMutation,
    changeResumeMutation,
    toggleFavoriteMutation,
    moveMutation,
    archiveMutation,
    unarchiveMutation,
    deleteMutation,
    completeCurrentStageMutation,
    addCommentMutation,
    checkMatchMutation,
  } = useJobDetailMutations({
    id,
    effectiveMatchResumeId,
    setFields,
    setNewComment,
    setMatchScore,
    setMatchScoreError,
    setIsPricingModalOpen,
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
      <JobDetailHeader
        job={job}
        isDirty={isDirty}
        canSave={!!editableFields.title.trim()}
        isSaving={updateMutation.isPending}
        onBack={() => navigate("/app/jobs")}
        onSave={handleSave}
        onDiscard={handleDiscard}
        onToggleFavorite={() => toggleFavoriteMutation.mutate()}
        isTogglingFavorite={toggleFavoriteMutation.isPending}
        onArchive={() => archiveMutation.mutate()}
        isArchiving={archiveMutation.isPending}
        onUnarchive={() => unarchiveMutation.mutate()}
        isUnarchiving={unarchiveMutation.isPending}
        onDelete={() => setIsDeleteConfirmOpen(true)}
        isDeleting={deleteMutation.isPending}
      />

      {/* Main info */}
      <JobInfoCard
        job={job}
        fields={editableFields}
        companies={companiesData?.items ?? []}
        onFieldChange={updateField}
      />

      {/* Pipeline */}
      <JobPipelineCard
        job={job}
        templates={templatesData?.items ?? []}
        uploadedResumes={uploadedResumes}
        builderResumes={builderResumesList}
        onMoveToStage={(stageTemplateId) =>
          moveMutation.mutate(stageTemplateId)
        }
        isMoving={moveMutation.isPending}
        onCompleteStage={(stageId) =>
          completeCurrentStageMutation.mutate(stageId)
        }
        isCompletingStage={completeCurrentStageMutation.isPending}
        onChangeResume={(value) => changeResumeMutation.mutate(value)}
        isChangingResume={changeResumeMutation.isPending}
      />

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
      <JobMatchScoreCard
        uploadedResumes={uploadedResumes}
        effectiveMatchResumeId={effectiveMatchResumeId}
        onSelectResume={(value) => setSelectedMatchResumeId(value)}
        onCheckMatch={() => checkMatchMutation.mutate()}
        isChecking={checkMatchMutation.isPending}
        matchScore={matchScore}
        matchScoreError={matchScoreError}
      />

      {/* Tags */}
      <JobTags jobId={id!} />

      {/* Comments */}
      <JobCommentsSection
        comments={jobComments}
        newComment={newComment}
        onChangeComment={(value) => setNewComment(value)}
        onAddComment={handleAddComment}
        isAdding={addCommentMutation.isPending}
      />

      {/* Reminders */}
      <JobReminders jobId={id!} />

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
