import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  analyticsService,
  type StageTimeAnalytics,
  type ResumeAnalytics,
  type SourceAnalytics,
} from "@/services/analyticsService";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/shared/ui/Card";
import { Skeleton } from "@/shared/ui/Skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import { Button } from "@/shared/ui/Button";
import { BarChart3, Clock, FileText, Globe, Share2 } from "lucide-react";
import { cn } from "@/shared/lib/utils";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { OverviewCards } from "@/features/analytics/components/OverviewCards";
import { FunnelVisualization } from "@/features/analytics/components/FunnelVisualization";
import { ShareStatsModal } from "@/features/sharing/components/ShareStatsModal";

// Stage Time Table Component
function StageTimeTable({
  data,
  isLoading,
}: {
  data?: StageTimeAnalytics;
  isLoading: boolean;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            {t("analytics.stageTime.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex gap-4">
                <Skeleton className="h-10 flex-1" />
                <Skeleton className="h-10 w-20" />
                <Skeleton className="h-10 w-20" />
                <Skeleton className="h-10 w-20" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data?.stages || data.stages.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            {t("analytics.stageTime.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<Clock className="h-12 w-12" />}
            title={t("analytics.stageTime.noData")}
            description={t("analytics.stageTime.noDataDescription")}
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Clock className="h-5 w-5" />
          {t("analytics.stageTime.title")}
        </CardTitle>
        <CardDescription>
          {t("analytics.stageTime.description")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {/* Mobile card view */}
        <div className="sm:hidden space-y-3">
          {data.stages.map((stage) => (
            <div
              key={stage.stage_name}
              className="rounded-lg border p-3 space-y-1.5"
            >
              <p className="font-medium">{stage.stage_name}</p>
              <div className="grid grid-cols-2 gap-1 text-sm">
                <span className="text-muted-foreground">
                  {t("analytics.stageTime.avgDays")}
                </span>
                <span className="text-right">{stage.avg_days}</span>
                <span className="text-muted-foreground">
                  {t("analytics.stageTime.minDays")}
                </span>
                <span className="text-right text-muted-foreground">
                  {stage.min_days}
                </span>
                <span className="text-muted-foreground">
                  {t("analytics.stageTime.maxDays")}
                </span>
                <span className="text-right text-muted-foreground">
                  {stage.max_days}
                </span>
                <span className="text-muted-foreground">
                  {t("analytics.stageTime.applications")}
                </span>
                <span className="text-right">{stage.applications_count}</span>
              </div>
            </div>
          ))}
        </div>
        {/* Desktop table view */}
        <div className="hidden sm:block overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.stageTime.stage")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.stageTime.avgDays")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.stageTime.minDays")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.stageTime.maxDays")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.stageTime.applications")}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.stages.map((stage) => (
                <tr
                  key={stage.stage_name}
                  className="border-b last:border-0 hover:bg-muted/50"
                >
                  <td className="py-3 px-2 font-medium">{stage.stage_name}</td>
                  <td className="py-3 px-2 text-right">{stage.avg_days}</td>
                  <td className="py-3 px-2 text-right text-muted-foreground">
                    {stage.min_days}
                  </td>
                  <td className="py-3 px-2 text-right text-muted-foreground">
                    {stage.max_days}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {stage.applications_count}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

// Resume Effectiveness Table Component
function ResumeEffectivenessTable({
  data,
  isLoading,
}: {
  data?: ResumeAnalytics;
  isLoading: boolean;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            {t("analytics.resumes.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex gap-4">
                <Skeleton className="h-10 flex-1" />
                <Skeleton className="h-10 w-16" />
                <Skeleton className="h-10 w-16" />
                <Skeleton className="h-10 w-16" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data?.resumes || data.resumes.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            {t("analytics.resumes.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<FileText className="h-12 w-12" />}
            title={t("analytics.resumes.noData")}
            description={t("analytics.resumes.noDataDescription")}
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          {t("analytics.resumes.title")}
        </CardTitle>
        <CardDescription>{t("analytics.resumes.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {/* Mobile card view */}
        <div className="sm:hidden space-y-3">
          {data.resumes.map((resume) => (
            <div
              key={resume.resume_id}
              className="rounded-lg border p-3 space-y-1.5"
            >
              <p className="font-medium">{resume.resume_title}</p>
              <div className="grid grid-cols-2 gap-1 text-sm">
                <span className="text-muted-foreground">
                  {t("analytics.resumes.applications")}
                </span>
                <span className="text-right">{resume.applications_count}</span>
                <span className="text-muted-foreground">
                  {t("analytics.resumes.responses")}
                </span>
                <span className="text-right">{resume.responses_count}</span>
                <span className="text-muted-foreground">
                  {t("analytics.resumes.interviews")}
                </span>
                <span className="text-right">{resume.interviews_count}</span>
                <span className="text-muted-foreground">
                  {t("analytics.resumes.responseRate")}
                </span>
                <span className="text-right">
                  <span
                    className={cn(
                      "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium",
                      resume.response_rate >= 50
                        ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                        : resume.response_rate >= 25
                          ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
                          : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
                    )}
                  >
                    {resume.response_rate}%
                  </span>
                </span>
              </div>
            </div>
          ))}
        </div>
        {/* Desktop table view */}
        <div className="hidden sm:block overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.resumes.resume")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.resumes.applications")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.resumes.responses")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.resumes.interviews")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.resumes.responseRate")}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.resumes.map((resume) => (
                <tr
                  key={resume.resume_id}
                  className="border-b last:border-0 hover:bg-muted/50"
                >
                  <td className="py-3 px-2 font-medium">
                    {resume.resume_title}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {resume.applications_count}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {resume.responses_count}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {resume.interviews_count}
                  </td>
                  <td className="py-3 px-2 text-right">
                    <span
                      className={cn(
                        "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium",
                        resume.response_rate >= 50
                          ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                          : resume.response_rate >= 25
                            ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
                            : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
                      )}
                    >
                      {resume.response_rate}%
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

// Source Analytics Table Component
function SourceAnalyticsTable({
  data,
  isLoading,
}: {
  data?: SourceAnalytics;
  isLoading: boolean;
}) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            {t("analytics.sources.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex gap-4">
                <Skeleton className="h-10 flex-1" />
                <Skeleton className="h-10 w-16" />
                <Skeleton className="h-10 w-16" />
                <Skeleton className="h-10 w-20" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data?.sources || data.sources.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            {t("analytics.sources.title")}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<Globe className="h-12 w-12" />}
            title={t("analytics.sources.noData")}
            description={t("analytics.sources.noDataDescription")}
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Globe className="h-5 w-5" />
          {t("analytics.sources.title")}
        </CardTitle>
        <CardDescription>{t("analytics.sources.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        {/* Mobile card view */}
        <div className="sm:hidden space-y-3">
          {data.sources.map((source) => (
            <div
              key={source.source_name}
              className="rounded-lg border p-3 space-y-1.5"
            >
              <p className="font-medium">{source.source_name}</p>
              <div className="grid grid-cols-2 gap-1 text-sm">
                <span className="text-muted-foreground">
                  {t("analytics.sources.applications")}
                </span>
                <span className="text-right">{source.applications_count}</span>
                <span className="text-muted-foreground">
                  {t("analytics.sources.responses")}
                </span>
                <span className="text-right">{source.responses_count}</span>
                <span className="text-muted-foreground">
                  {t("analytics.sources.conversionRate")}
                </span>
                <span className="text-right">
                  <span
                    className={cn(
                      "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium",
                      source.conversion_rate >= 50
                        ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                        : source.conversion_rate >= 25
                          ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
                          : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
                    )}
                  >
                    {source.conversion_rate}%
                  </span>
                </span>
              </div>
            </div>
          ))}
        </div>
        {/* Desktop table view */}
        <div className="hidden sm:block overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.sources.source")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.sources.applications")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.sources.responses")}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t("analytics.sources.conversionRate")}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.sources.map((source) => (
                <tr
                  key={source.source_name}
                  className="border-b last:border-0 hover:bg-muted/50"
                >
                  <td className="py-3 px-2 font-medium">
                    {source.source_name}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {source.applications_count}
                  </td>
                  <td className="py-3 px-2 text-right">
                    {source.responses_count}
                  </td>
                  <td className="py-3 px-2 text-right">
                    <span
                      className={cn(
                        "inline-flex items-center px-2 py-1 rounded-full text-xs font-medium",
                        source.conversion_rate >= 50
                          ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                          : source.conversion_rate >= 25
                            ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300"
                            : "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300",
                      )}
                    >
                      {source.conversion_rate}%
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

// Main Analytics Page Component
export default function Analytics() {
  const { t } = useTranslation();
  usePageMeta({ titleKey: "analytics.title", noindex: true });
  const [shareModalOpen, setShareModalOpen] = useState(false);

  const overviewQuery = useQuery({
    queryKey: ["analytics", "overview"],
    queryFn: () => analyticsService.getOverview(),
  });

  const funnelQuery = useQuery({
    queryKey: ["analytics", "funnel"],
    queryFn: () => analyticsService.getFunnel(),
  });

  const stageTimeQuery = useQuery({
    queryKey: ["analytics", "stageTime"],
    queryFn: () => analyticsService.getStageTime(),
  });

  const resumeQuery = useQuery({
    queryKey: ["analytics", "resumes"],
    queryFn: () => analyticsService.getResumeEffectiveness(),
  });

  const sourceQuery = useQuery({
    queryKey: ["analytics", "sources"],
    queryFn: () => analyticsService.getSourceAnalytics(),
  });

  const isAllLoading =
    overviewQuery.isLoading &&
    funnelQuery.isLoading &&
    stageTimeQuery.isLoading &&
    resumeQuery.isLoading &&
    sourceQuery.isLoading;

  const hasAnyError =
    overviewQuery.isError ||
    funnelQuery.isError ||
    stageTimeQuery.isError ||
    resumeQuery.isError ||
    sourceQuery.isError;

  // Check if there's no data at all
  const hasNoData =
    !overviewQuery.isLoading && overviewQuery.data?.total_applications === 0;

  if (hasAnyError && !isAllLoading) {
    const error =
      overviewQuery.error ||
      funnelQuery.error ||
      stageTimeQuery.error ||
      resumeQuery.error ||
      sourceQuery.error;

    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-bold">{t("analytics.title")}</h1>
        </div>
        <ErrorState
          message={(error as Error)?.message || t("analytics.error")}
          onRetry={() => {
            overviewQuery.refetch();
            funnelQuery.refetch();
            stageTimeQuery.refetch();
            resumeQuery.refetch();
            sourceQuery.refetch();
          }}
        />
      </div>
    );
  }

  if (hasNoData) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-bold">{t("analytics.title")}</h1>
        </div>
        <EmptyState
          icon={<BarChart3 className="h-16 w-16" />}
          title={t("analytics.noData")}
          description={t("analytics.noDataDescription")}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-8 w-8 text-primary" />
          <div>
            <h1 className="text-3xl font-bold">{t("analytics.title")}</h1>
            <p className="text-muted-foreground">
              {t("analytics.description")}
            </p>
          </div>
        </div>
        <Button
          variant="outline"
          onClick={() => setShareModalOpen(true)}
          className="self-start sm:self-auto"
        >
          <Share2 className="h-4 w-4 mr-2" />
          {t("sharing.shareButton")}
        </Button>
      </div>
      <ShareStatsModal
        open={shareModalOpen}
        onOpenChange={setShareModalOpen}
        overview={overviewQuery.data}
        funnel={funnelQuery.data}
      />

      {/* Overview Cards */}
      <OverviewCards
        data={overviewQuery.data}
        isLoading={overviewQuery.isLoading}
      />

      {/* Funnel Visualization */}
      <FunnelVisualization
        data={funnelQuery.data}
        isLoading={funnelQuery.isLoading}
      />

      {/* Two-column grid for tables */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Stage Time Table */}
        <StageTimeTable
          data={stageTimeQuery.data}
          isLoading={stageTimeQuery.isLoading}
        />

        {/* Resume Effectiveness Table */}
        <ResumeEffectivenessTable
          data={resumeQuery.data}
          isLoading={resumeQuery.isLoading}
        />
      </div>

      {/* Source Analytics Table */}
      <SourceAnalyticsTable
        data={sourceQuery.data}
        isLoading={sourceQuery.isLoading}
      />
    </div>
  );
}
