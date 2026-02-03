import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { 
  analyticsService,
  type OverviewAnalytics,
  type FunnelAnalytics,
  type StageTimeAnalytics,
  type ResumeAnalytics,
  type SourceAnalytics,
} from '@/services/analyticsService';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/shared/ui/Card';
import { Skeleton } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { 
  BarChart3, 
  TrendingUp,
  Clock,
  FileText,
  Globe,
  Activity,
  CheckCircle,
  ArrowRight,
  Briefcase,
} from 'lucide-react';
import { cn } from '@/shared/lib/utils';

// Overview Cards Component
function OverviewCards({ data, isLoading }: { data?: OverviewAnalytics; isLoading: boolean }) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i}>
            <CardContent className="p-6">
              <Skeleton className="h-4 w-24 mb-2" />
              <Skeleton className="h-8 w-16" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (!data) return null;

  const cards = [
    {
      title: t('analytics.overview.totalApplications'),
      value: data.total_applications,
      icon: Briefcase,
      color: 'text-blue-500',
    },
    {
      title: t('analytics.overview.activeApplications'),
      value: data.active_applications,
      icon: Activity,
      color: 'text-green-500',
    },
    {
      title: t('analytics.overview.closedApplications'),
      value: data.closed_applications,
      icon: CheckCircle,
      color: 'text-gray-500',
    },
    {
      title: t('analytics.overview.responseRate'),
      value: `${data.response_rate}%`,
      icon: TrendingUp,
      color: 'text-purple-500',
    },
    {
      title: t('analytics.overview.avgResponseTime'),
      value: data.avg_days_to_first_response > 0 
        ? `${data.avg_days_to_first_response} ${t('analytics.days')}`
        : '-',
      icon: Clock,
      color: 'text-orange-500',
    },
  ];

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
      {cards.map((card, index) => {
        const Icon = card.icon;
        return (
          <Card key={index}>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">{card.title}</p>
                  <p className="text-2xl font-bold mt-1">{card.value}</p>
                </div>
                <Icon className={cn('h-8 w-8', card.color)} />
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}

// Funnel Visualization Component
function FunnelVisualization({ data, isLoading }: { data?: FunnelAnalytics; isLoading: boolean }) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            {t('analytics.funnel.title')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4">
                <Skeleton className="h-12 flex-1" />
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
            <TrendingUp className="h-5 w-5" />
            {t('analytics.funnel.title')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<TrendingUp className="h-12 w-12" />}
            title={t('analytics.funnel.noData')}
            description={t('analytics.funnel.noDataDescription')}
          />
        </CardContent>
      </Card>
    );
  }

  const maxCount = Math.max(...data.stages.map(s => s.count), 1);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5" />
          {t('analytics.funnel.title')}
        </CardTitle>
        <CardDescription>{t('analytics.funnel.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {data.stages.map((stage, index) => {
            const widthPercent = (stage.count / maxCount) * 100;
            return (
              <div key={stage.stage_name} className="space-y-1">
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium">{stage.stage_name}</span>
                  <div className="flex items-center gap-4 text-muted-foreground">
                    <span>{stage.count} {t('analytics.applications')}</span>
                    {index > 0 && (
                      <>
                        <span className="text-green-600">
                          {stage.conversion_rate}% {t('analytics.funnel.converted')}
                        </span>
                        <span className="text-red-500">
                          {stage.drop_off_rate}% {t('analytics.funnel.dropOff')}
                        </span>
                      </>
                    )}
                  </div>
                </div>
                <div className="h-8 bg-muted rounded-md overflow-hidden">
                  <div 
                    className="h-full bg-primary/80 rounded-md transition-all duration-500 flex items-center justify-end pr-2"
                    style={{ width: `${Math.max(widthPercent, 5)}%` }}
                  >
                    {widthPercent > 15 && (
                      <span className="text-xs text-primary-foreground font-medium">
                        {stage.count}
                      </span>
                    )}
                  </div>
                </div>
                {index < data.stages.length - 1 && (
                  <div className="flex justify-center py-1">
                    <ArrowRight className="h-4 w-4 text-muted-foreground rotate-90" />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// Stage Time Table Component
function StageTimeTable({ data, isLoading }: { data?: StageTimeAnalytics; isLoading: boolean }) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            {t('analytics.stageTime.title')}
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
            {t('analytics.stageTime.title')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<Clock className="h-12 w-12" />}
            title={t('analytics.stageTime.noData')}
            description={t('analytics.stageTime.noDataDescription')}
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
          {t('analytics.stageTime.title')}
        </CardTitle>
        <CardDescription>{t('analytics.stageTime.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.stageTime.stage')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.stageTime.avgDays')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.stageTime.minDays')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.stageTime.maxDays')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.stageTime.applications')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.stages.map((stage) => (
                <tr key={stage.stage_name} className="border-b last:border-0 hover:bg-muted/50">
                  <td className="py-3 px-2 font-medium">{stage.stage_name}</td>
                  <td className="py-3 px-2 text-right">{stage.avg_days}</td>
                  <td className="py-3 px-2 text-right text-muted-foreground">{stage.min_days}</td>
                  <td className="py-3 px-2 text-right text-muted-foreground">{stage.max_days}</td>
                  <td className="py-3 px-2 text-right">{stage.applications_count}</td>
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
function ResumeEffectivenessTable({ data, isLoading }: { data?: ResumeAnalytics; isLoading: boolean }) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            {t('analytics.resumes.title')}
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
            {t('analytics.resumes.title')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<FileText className="h-12 w-12" />}
            title={t('analytics.resumes.noData')}
            description={t('analytics.resumes.noDataDescription')}
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
          {t('analytics.resumes.title')}
        </CardTitle>
        <CardDescription>{t('analytics.resumes.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.resumes.resume')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.resumes.applications')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.resumes.responses')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.resumes.interviews')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.resumes.responseRate')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.resumes.map((resume) => (
                <tr key={resume.resume_id} className="border-b last:border-0 hover:bg-muted/50">
                  <td className="py-3 px-2 font-medium">{resume.resume_title}</td>
                  <td className="py-3 px-2 text-right">{resume.applications_count}</td>
                  <td className="py-3 px-2 text-right">{resume.responses_count}</td>
                  <td className="py-3 px-2 text-right">{resume.interviews_count}</td>
                  <td className="py-3 px-2 text-right">
                    <span className={cn(
                      'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                      resume.response_rate >= 50 ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' :
                      resume.response_rate >= 25 ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300' :
                      'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                    )}>
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
function SourceAnalyticsTable({ data, isLoading }: { data?: SourceAnalytics; isLoading: boolean }) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-5 w-5" />
            {t('analytics.sources.title')}
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
            {t('analytics.sources.title')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={<Globe className="h-12 w-12" />}
            title={t('analytics.sources.noData')}
            description={t('analytics.sources.noDataDescription')}
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
          {t('analytics.sources.title')}
        </CardTitle>
        <CardDescription>{t('analytics.sources.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.sources.source')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.sources.applications')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.sources.responses')}
                </th>
                <th className="text-right py-3 px-2 text-sm font-medium text-muted-foreground">
                  {t('analytics.sources.conversionRate')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.sources.map((source) => (
                <tr key={source.source_name} className="border-b last:border-0 hover:bg-muted/50">
                  <td className="py-3 px-2 font-medium">{source.source_name}</td>
                  <td className="py-3 px-2 text-right">{source.applications_count}</td>
                  <td className="py-3 px-2 text-right">{source.responses_count}</td>
                  <td className="py-3 px-2 text-right">
                    <span className={cn(
                      'inline-flex items-center px-2 py-1 rounded-full text-xs font-medium',
                      source.conversion_rate >= 50 ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' :
                      source.conversion_rate >= 25 ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300' :
                      'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                    )}>
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

  const overviewQuery = useQuery({
    queryKey: ['analytics', 'overview'],
    queryFn: () => analyticsService.getOverview(),
  });

  const funnelQuery = useQuery({
    queryKey: ['analytics', 'funnel'],
    queryFn: () => analyticsService.getFunnel(),
  });

  const stageTimeQuery = useQuery({
    queryKey: ['analytics', 'stageTime'],
    queryFn: () => analyticsService.getStageTime(),
  });

  const resumeQuery = useQuery({
    queryKey: ['analytics', 'resumes'],
    queryFn: () => analyticsService.getResumeEffectiveness(),
  });

  const sourceQuery = useQuery({
    queryKey: ['analytics', 'sources'],
    queryFn: () => analyticsService.getSourceAnalytics(),
  });

  const isAllLoading = overviewQuery.isLoading && funnelQuery.isLoading && 
    stageTimeQuery.isLoading && resumeQuery.isLoading && sourceQuery.isLoading;

  const hasAnyError = overviewQuery.isError || funnelQuery.isError || 
    stageTimeQuery.isError || resumeQuery.isError || sourceQuery.isError;

  // Check if there's no data at all
  const hasNoData = !overviewQuery.isLoading && 
    overviewQuery.data?.total_applications === 0;

  if (hasAnyError && !isAllLoading) {
    const error = overviewQuery.error || funnelQuery.error || 
      stageTimeQuery.error || resumeQuery.error || sourceQuery.error;
    
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-bold">{t('analytics.title')}</h1>
        </div>
        <ErrorState
          message={(error as Error)?.message || t('analytics.error')}
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
          <h1 className="text-3xl font-bold">{t('analytics.title')}</h1>
        </div>
        <EmptyState
          icon={<BarChart3 className="h-16 w-16" />}
          title={t('analytics.noData')}
          description={t('analytics.noDataDescription')}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <BarChart3 className="h-8 w-8 text-primary" />
        <div>
          <h1 className="text-3xl font-bold">{t('analytics.title')}</h1>
          <p className="text-muted-foreground">{t('analytics.description')}</p>
        </div>
      </div>

      {/* Overview Cards */}
      <OverviewCards data={overviewQuery.data} isLoading={overviewQuery.isLoading} />

      {/* Funnel Visualization */}
      <FunnelVisualization data={funnelQuery.data} isLoading={funnelQuery.isLoading} />

      {/* Two-column grid for tables */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Stage Time Table */}
        <StageTimeTable data={stageTimeQuery.data} isLoading={stageTimeQuery.isLoading} />

        {/* Resume Effectiveness Table */}
        <ResumeEffectivenessTable data={resumeQuery.data} isLoading={resumeQuery.isLoading} />
      </div>

      {/* Source Analytics Table */}
      <SourceAnalyticsTable data={sourceQuery.data} isLoading={sourceQuery.isLoading} />
    </div>
  );
}
