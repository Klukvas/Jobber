import { useState, useMemo } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import { resumesService } from "@/services/resumesService";
import { matchScoreService } from "@/services/matchScoreService";
import { ApiError } from "@/services/api";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { SkeletonDetail } from "@/shared/ui/Skeleton";
import { ErrorState } from "@/shared/ui/ErrorState";
import { MatchScoreCard } from "@/features/applications/components/MatchScoreCard";
import { CompanySelectWithQuickAdd } from "@/features/jobs/components/CompanySelectWithQuickAdd";
import { PricingModal } from "@/features/subscription/components/PricingModal";
import { companiesService } from "@/services/companiesService";
import {
  ArrowLeft,
  Calendar,
  ExternalLink,
  Save,
  Archive,
  Briefcase,
  Sparkles,
  Loader2,
  Heart,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
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

export default function JobDetail() {
  usePageMeta({ titleKey: "jobs.details", noindex: true });
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [fields, setFields] = useState<EditableFields | null>(null);
  const [selectedResumeId, setSelectedResumeId] = useState("");
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

  const checkMatchMutation = useMutation({
    mutationFn: () => matchScoreService.checkMatch(id!, selectedResumeId),
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
          setMatchScoreError(t("applications.matchScore.noDescription"));
        } else if (err.code === "RESUME_FILE_EMPTY") {
          setMatchScoreError(t("applications.matchScore.noResumeFile"));
        } else if (err.code === "AI_NOT_CONFIGURED") {
          setMatchScoreError(t("applications.matchScore.aiNotAvailable"));
        } else {
          setMatchScoreError(t("applications.matchScore.error"));
        }
      } else {
        setMatchScoreError(t("applications.matchScore.error"));
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

  const isValidUrl = (url: string) => {
    try {
      const protocol = new URL(url).protocol;
      return protocol === "http:" || protocol === "https:";
    } catch {
      return false;
    }
  };

  const resumes = resumesData?.items ?? [];

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
          message={error?.message || t("errors.notFound")}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate("/app/jobs")}>
          <ArrowLeft className="h-4 w-4" />
          {t("jobs.backToJobs")}
        </Button>
        <div className="flex gap-2">
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
          {job.status === "active" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => archiveMutation.mutate(job.id)}
              disabled={archiveMutation.isPending}
            >
              <Archive className="h-4 w-4 mr-2" />
              {t("jobs.archive")}
            </Button>
          )}
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
            <span
              className={`inline-flex items-center rounded-full text-sm px-2.5 py-1 font-medium shrink-0 ${
                job.status === "active"
                  ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                  : "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400"
              }`}
            >
              {job.status === "active"
                ? t("common.active")
                : t("jobs.statusArchived")}
            </span>
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
              })}
            </span>
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
        <CardContent>
          <Textarea
            value={editableFields.notes}
            onChange={(e) => updateField("notes", e.target.value)}
            placeholder={t("jobs.notesPlaceholder")}
            className="min-h-[80px]"
          />
        </CardContent>
      </Card>

      {/* Check Match */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            {t("applications.matchScore.checkMatch")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="resume-select">{t("jobs.selectResume")}</Label>
              <select
                id="resume-select"
                value={selectedResumeId}
                onChange={(e) => setSelectedResumeId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">{t("jobs.selectResume")}</option>
                {resumes.map((resume) => (
                  <option key={resume.id} value={resume.id}>
                    {resume.title}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={() => checkMatchMutation.mutate()}
              disabled={!selectedResumeId || checkMatchMutation.isPending}
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
          </div>

          {matchScoreError && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
              {matchScoreError}
            </div>
          )}
        </CardContent>
      </Card>

      {matchScore && <MatchScoreCard data={matchScore} />}

      {/* Related applications */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Briefcase className="h-5 w-5" />
            {t("jobs.relatedApplications")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {job.applications_count > 0 ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">
                {t("jobs.applicationsCount", {
                  count: job.applications_count,
                })}
              </p>
              <Link
                to="/app/applications"
                className="inline-flex items-center text-sm text-primary hover:underline"
              >
                {t("jobs.viewApplications")}
                <ExternalLink className="h-3 w-3 ml-1" />
              </Link>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground italic">
              {t("jobs.noRelatedApplications")}
            </p>
          )}
        </CardContent>
      </Card>

      <PricingModal
        open={isPricingModalOpen}
        onOpenChange={setIsPricingModalOpen}
      />
    </div>
  );
}
