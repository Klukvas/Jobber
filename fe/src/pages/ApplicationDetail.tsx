import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { applicationsService } from "@/services/applicationsService";
import { commentsService } from "@/services/commentsService";
import { matchScoreService } from "@/services/matchScoreService";
import { ApiError } from "@/services/api";
import { showErrorNotification } from "@/shared/lib/notifications";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { SkeletonDetail } from "@/shared/ui/Skeleton";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  ArrowLeft,
  Calendar,
  Plus,
  MessageSquarePlus,
  Edit,
  Sparkles,
  Loader2,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { Timeline } from "@/features/applications/components/Timeline";
import { AddStageModal } from "@/features/applications/modals/AddStageModal";
import { UpdateApplicationStatusModal } from "@/features/applications/modals/UpdateApplicationStatusModal";
import { MatchScoreCard } from "@/features/applications/components/MatchScoreCard";
import { Textarea } from "@/shared/ui/Textarea";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import type { MatchScoreResponse } from "@/shared/types/api";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import type { ApplicationStatus } from "@/shared/types/api";

export default function ApplicationDetail() {
  usePageMeta({ titleKey: "applications.details", noindex: true });
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isAddStageModalOpen, setIsAddStageModalOpen] = useState(false);
  const [isUpdateStatusModalOpen, setIsUpdateStatusModalOpen] = useState(false);
  const [newComment, setNewComment] = useState("");
  const [matchScore, setMatchScore] = useState<MatchScoreResponse | null>(null);
  const [isPricingModalOpen, setIsPricingModalOpen] = useState(false);

  const {
    data: application,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["application", id],
    queryFn: () => applicationsService.getById(id!),
    enabled: !!id,
  });

  const { data: stages } = useQuery({
    queryKey: ["application-stages", id],
    queryFn: () => applicationsService.listStages(id!),
    enabled: !!id,
  });

  // Comments are now provided by the backend (pre-split)
  const applicationComments = application?.application_comments || [];

  const addCommentMutation = useMutation({
    mutationFn: (content: string) =>
      commentsService.create({
        application_id: id!,
        content,
      }),
    onSuccess: () => {
      // Invalidate application query to refresh embedded comments
      queryClient.invalidateQueries({ queryKey: ["application", id] });
      setNewComment("");
    },
  });

  const checkMatchMutation = useMutation({
    mutationFn: () => {
      if (!application?.job?.id) {
        return Promise.reject(new Error(t("applications.matchScore.error")));
      }
      if (!application?.resume?.id) {
        return Promise.reject(new Error(t("applications.matchScore.noResume")));
      }
      return matchScoreService.checkMatch(
        application.job.id,
        application.resume.id,
      );
    },
    onSuccess: (data) => {
      setMatchScore(data);
      setIsPricingModalOpen(false);
    },
    onError: (error: Error) => {
      if (error instanceof ApiError) {
        if (error.code === "PLAN_LIMIT_REACHED") {
          setIsPricingModalOpen(true);
          queryClient.invalidateQueries({ queryKey: ["subscription"] });
        } else if (error.code === "JOB_DESCRIPTION_EMPTY") {
          showErrorNotification(t("applications.matchScore.noDescription"));
        } else if (error.code === "RESUME_FILE_EMPTY") {
          showErrorNotification(t("applications.matchScore.noResumeFile"));
        } else if (error.code === "AI_NOT_CONFIGURED") {
          showErrorNotification(t("applications.matchScore.aiNotAvailable"));
        } else {
          showErrorNotification(t("applications.matchScore.error"));
        }
      } else {
        showErrorNotification(
          error.message || t("applications.matchScore.error"),
        );
      }
    },
  });

  const handleAddComment = (e: React.FormEvent) => {
    e.preventDefault();
    if (newComment.trim()) {
      addCommentMutation.mutate(newComment.trim());
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate("/app/applications")}>
          <ArrowLeft className="h-4 w-4" />
          {t("common.back")}
        </Button>
        <SkeletonDetail />
      </div>
    );
  }

  if (isError || !application) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate("/app/applications")}>
          <ArrowLeft className="h-4 w-4" />
          {t("common.back")}
        </Button>
        <ErrorState
          message={error?.message || t("applications.notFound")}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate("/app/applications")}>
          <ArrowLeft className="h-4 w-4" />
          {t("common.back")}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{application.name || t("applications.details")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-sm text-muted-foreground">
              {t("applications.applicationName")}
            </p>
            <p className="font-semibold text-lg">
              {application.name || t("applications.untitled")}
            </p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">
              {t("applications.job")}
            </p>
            {application.job ? (
              <Link
                to={`/app/jobs/${application.job.id}`}
                className="font-medium hover:underline text-primary"
              >
                {application.job.title}
              </Link>
            ) : (
              <p className="font-medium">{t("applications.unknownJob")}</p>
            )}
          </div>
          <div>
            <p className="text-sm text-muted-foreground">
              {t("applications.resume")}
            </p>
            <p className="font-medium">
              {application.resume?.name || t("applications.unknownResume")}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">
              {t("applications.applied")}{" "}
              {formatDistanceToNow(new Date(application.applied_at), {
                addSuffix: true,
                locale: dateLocale,
              })}
            </span>
          </div>
          <div>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-sm text-muted-foreground">
                  {t("applications.status")}
                </p>
                <StatusBadge status={application.status as ApplicationStatus} />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => checkMatchMutation.mutate()}
                  disabled={
                    checkMatchMutation.isPending || !application?.job?.id
                  }
                >
                  {checkMatchMutation.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <Sparkles className="h-4 w-4 mr-2" />
                  )}
                  {checkMatchMutation.isPending
                    ? t("applications.matchScore.checking")
                    : t("applications.matchScore.checkMatch")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setIsUpdateStatusModalOpen(true)}
                >
                  <Edit className="h-4 w-4 mr-2" />
                  {t("applications.changeStatus")}
                </Button>
              </div>
            </div>
          </div>

          <div className="mt-6 border-t pt-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold">
                {t("applications.comments")}
              </h3>
            </div>

            {applicationComments.length > 0 && (
              <div className="space-y-3 mb-4">
                {applicationComments.map((comment) => (
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
                placeholder={t("applications.commentPlaceholder")}
                className="flex-1"
                rows={3}
              />
              <Button
                type="submit"
                size="sm"
                disabled={!newComment.trim() || addCommentMutation.isPending}
              >
                <MessageSquarePlus className="h-4 w-4 mr-2" />
                {t("applications.addComment")}
              </Button>
            </form>
          </div>
        </CardContent>
      </Card>

      {matchScore && <MatchScoreCard data={matchScore} />}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle>{t("applications.timeline")}</CardTitle>
          <Button size="sm" onClick={() => setIsAddStageModalOpen(true)}>
            <Plus className="h-4 w-4" />
            {t("applications.addNewStage")}
          </Button>
        </CardHeader>
        <CardContent>
          <Timeline
            stages={stages || []}
            applicationId={id!}
            stageComments={application?.stage_comments || []}
          />
        </CardContent>
      </Card>

      <AddStageModal
        open={isAddStageModalOpen}
        onOpenChange={setIsAddStageModalOpen}
        applicationId={id!}
      />

      {application && (
        <UpdateApplicationStatusModal
          open={isUpdateStatusModalOpen}
          onOpenChange={setIsUpdateStatusModalOpen}
          applicationId={id!}
          currentStatus={application.status}
        />
      )}

      <PricingModal
        open={isPricingModalOpen}
        onOpenChange={setIsPricingModalOpen}
      />
    </div>
  );
}
