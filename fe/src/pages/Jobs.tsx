import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { formatDistanceToNow } from 'date-fns';
import { jobsService } from '@/services/jobsService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonList } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { Plus, Briefcase, ExternalLink, Building2, MoreVertical, Edit, Archive, Calendar, FileText, ArrowUpDown } from 'lucide-react';
import { CreateJobModal } from '@/features/jobs/modals/CreateJobModal';
import type { JobDTO } from '@/shared/types/api';
import { showSuccessNotification, showErrorNotification } from '@/shared/lib/notifications';

type SortField = 'created_at' | 'title' | 'company_name';
type SortDir = 'asc' | 'desc';

export default function Jobs() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<JobDTO | undefined>(undefined);
  const [sortField, setSortField] = useState<SortField>('created_at');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);

  // Close context menu when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setOpenMenuId(null);
    if (openMenuId) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['jobs', sortField, sortDir],
    queryFn: () => jobsService.list({ 
      limit: 100, 
      offset: 0,
      status: 'active',
      sort: `${sortField}:${sortDir}`,
    }),
  });

  const archiveMutation = useMutation({
    mutationFn: jobsService.archive,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      showSuccessNotification(t('jobs.archiveSuccess'));
    },
    onError: () => {
      showErrorNotification(t('jobs.archiveError'));
    },
  });

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
    } else {
      setSortField(field);
      setSortDir('desc');
    }
  };

  const handleEdit = (job: JobDTO) => {
    setEditingJob(job);
    setIsCreateModalOpen(true);
    setOpenMenuId(null);
  };

  const handleArchive = (jobId: string) => {
    archiveMutation.mutate(jobId);
    setOpenMenuId(null);
  };

  const handleModalClose = () => {
    setIsCreateModalOpen(false);
    setEditingJob(undefined);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t('jobs.title')}</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t('jobs.title')}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const jobs = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('jobs.title')}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t('jobs.create')}
        </Button>
      </div>

      {jobs.length === 0 ? (
        <EmptyState
          icon={<Briefcase className="h-12 w-12" />}
          title={t('jobs.emptyTitle')}
          description={t('jobs.emptyDescription')}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t('jobs.createFirstJob')}
            </Button>
          }
        />
      ) : (
        <>
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">{t('jobs.sortBy')}</span>
            <Button
              variant={sortField === 'created_at' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('created_at')}
            >
              <Calendar className="h-3 w-3 mr-1" />
              {t('jobs.sortCreatedDate')}
              {sortField === 'created_at' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortField === 'title' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('title')}
            >
              <FileText className="h-3 w-3 mr-1" />
              {t('jobs.sortJobTitle')}
              {sortField === 'title' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortField === 'company_name' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('company_name')}
            >
              <Building2 className="h-3 w-3 mr-1" />
              {t('jobs.sortCompanyName')}
              {sortField === 'company_name' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {jobs.map((job) => (
              <Card key={job.id} className="relative group">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg flex-1">{job.title}</CardTitle>
                    <div className="relative">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setOpenMenuId(openMenuId === job.id ? null : job.id);
                        }}
                        className="p-1 rounded-md hover:bg-accent transition-colors opacity-0 group-hover:opacity-100"
                        aria-label="Job actions"
                      >
                        <MoreVertical className="h-4 w-4" />
                      </button>
                      {openMenuId === job.id && (
                        <div className="absolute right-0 mt-1 w-40 bg-popover border rounded-md shadow-lg z-10">
                          <button
                            onClick={() => handleEdit(job)}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <Edit className="h-4 w-4" />
                            {t('common.edit')}
                          </button>
                          <button
                            onClick={() => handleArchive(job.id)}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <Archive className="h-4 w-4" />
                            {t('jobs.archive')}
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2">
                  {job.company_name && (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Building2 className="h-4 w-4" />
                      <span>{job.company_name}</span>
                    </div>
                  )}
                  {job.url && (
                    <div className="flex items-center gap-2 text-sm">
                      <a
                        href={job.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center gap-1 text-primary hover:underline"
                      >
                        {t('jobs.viewPosting')}
                        <ExternalLink className="h-3 w-3" />
                      </a>
                    </div>
                  )}
                  {job.source && (
                    <div className="text-sm text-muted-foreground">
                      {t('jobs.source')}: {job.source}
                    </div>
                  )}
                  <div className="text-sm text-muted-foreground pt-2 border-t space-y-1">
                    <div className="flex items-center gap-2">
                      <Calendar className="h-3.5 w-3.5" />
                      <span>
                        {t('jobs.createdDate')}{' '}
                        {formatDistanceToNow(new Date(job.created_at), {
                          addSuffix: true,
                        })}
                      </span>
                    </div>
                    <div>
                      {job.applications_count > 0 
                        ? t('jobs.applicationsCount', { count: job.applications_count })
                        : t('jobs.noApplications')}
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      <CreateJobModal
        open={isCreateModalOpen}
        onOpenChange={handleModalClose}
        job={editingJob}
      />
    </div>
  );
}
