import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { applicationsService } from '@/services/applicationsService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonList } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { StatusBadge } from '@/shared/ui/StatusBadge';
import { 
  Plus, 
  Briefcase, 
  Building2, 
  Calendar, 
  Clock,
  MoreVertical,
  MessageSquare,
  Archive,
  GitBranch,
  ArrowUpDown,
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { CreateApplicationModal } from '@/features/applications/modals/CreateApplicationModal';
import { AddCommentModal } from '@/features/applications/modals/AddCommentModal';
import { AddStageModal } from '@/features/applications/modals/AddStageModal';
import { UpdateApplicationStatusModal } from '@/features/applications/modals/UpdateApplicationStatusModal';
import { usePageTitle } from '@/shared/lib/usePageTitle';
import type { ApplicationDTO } from '@/shared/types/api';

type SortBy = 'last_activity' | 'status' | 'applied_at';
type SortDir = 'asc' | 'desc';

export default function Applications() {
  const { t } = useTranslation();
  usePageTitle('applications.title');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [page, setPage] = useState(0);
  const [sortBy, setSortBy] = useState<SortBy>('last_activity');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [activeQuickAction, setActiveQuickAction] = useState<{
    type: 'comment' | 'stage' | 'archive';
    application: ApplicationDTO;
  } | null>(null);
  const pageSize = 20;

  // Close context menu when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setOpenMenuId(null);
    if (openMenuId) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['applications', page, sortBy, sortDir],
    queryFn: () =>
      applicationsService.list({
        limit: pageSize,
        offset: page * pageSize,
        sort_by: sortBy,
        sort_dir: sortDir,
      }),
  });

  const handleQuickAction = (
    type: 'comment' | 'stage' | 'archive',
    application: ApplicationDTO
  ) => {
    setActiveQuickAction({ type, application });
    setOpenMenuId(null);
  };

  // Removed unused handleArchive function - status changes handled by UpdateApplicationStatusModal

  const toggleSort = (field: SortBy) => {
    if (sortBy === field) {
      setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
    } else {
      setSortBy(field);
      setSortDir('desc');
    }
    setPage(0);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t('applications.title')}</h1>
        </div>
        <SkeletonList count={5} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t('applications.title')}</h1>
        <ErrorState
          message={error.message}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const applications = data?.items || [];
  const pagination = data?.pagination;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('applications.title')}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t('applications.create')}
        </Button>
      </div>

      {applications.length === 0 ? (
        <EmptyState
          icon={<Briefcase className="h-12 w-12" />}
          title={t('applications.noApplications')}
          description={t('applications.createFirst')}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t('applications.create')}
            </Button>
          }
        />
      ) : (
        <>
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">Sort by:</span>
            <Button
              variant={sortBy === 'last_activity' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('last_activity')}
            >
              <Clock className="h-3 w-3 mr-1" />
              Last Activity
              {sortBy === 'last_activity' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortBy === 'status' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('status')}
            >
              Status
              {sortBy === 'status' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortBy === 'applied_at' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('applied_at')}
            >
              <Calendar className="h-3 w-3 mr-1" />
              Applied Date
              {sortBy === 'applied_at' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
          </div>

          {/* Application Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {applications.map((application) => (
              <div key={application.id} className="relative">
                <Link to={`/app/applications/${application.id}`}>
                  <Card className="transition-all hover:shadow-md h-full group">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between gap-2">
                        {/* Primary: Application Title */}
                        <CardTitle className="text-xl font-bold leading-tight mb-2 flex-1">
                          {application.name}
                        </CardTitle>
                        <div className="relative" onClick={(e) => e.preventDefault()}>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              e.preventDefault();
                              setOpenMenuId(openMenuId === application.id ? null : application.id);
                            }}
                            className="p-1 rounded-md hover:bg-accent transition-colors opacity-0 group-hover:opacity-100"
                            aria-label="Application actions"
                          >
                            <MoreVertical className="h-4 w-4" />
                          </button>
                          {openMenuId === application.id && (
                            <div className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10">
                              <button
                                onClick={(e) => {
                                  e.preventDefault();
                                  e.stopPropagation();
                                  handleQuickAction('comment', application);
                                }}
                                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                              >
                                <MessageSquare className="h-4 w-4" />
                                Add Comment
                              </button>
                              <button
                                onClick={(e) => {
                                  e.preventDefault();
                                  e.stopPropagation();
                                  handleQuickAction('stage', application);
                                }}
                                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                              >
                                <GitBranch className="h-4 w-4" />
                                Add Stage
                              </button>
                              <button
                                onClick={(e) => {
                                  e.preventDefault();
                                  e.stopPropagation();
                                  handleQuickAction('archive', application);
                                }}
                                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                              >
                                <Archive className="h-4 w-4" />
                                Change Status
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                      
                      {/* Secondary: Company and Job */}
                      <div className="space-y-1">
                        {application.job?.company?.name && (
                          <div className="flex items-center gap-2 text-base font-medium text-foreground">
                            <Building2 className="h-4 w-4 text-muted-foreground" />
                            <span>{application.job.company.name}</span>
                          </div>
                        )}
                        <div className="flex items-center gap-2 text-base text-foreground">
                          <Briefcase className="h-4 w-4 text-muted-foreground" />
                          <span>{application.job?.title || 'Unknown Job'}</span>
                        </div>
                      </div>
                    </CardHeader>
                    
                    <CardContent className="space-y-3 pt-0">
                      {/* Status - Dominant Signal */}
                      <div className="flex items-center justify-between">
                        <StatusBadge status={application.status} size="lg" />
                      </div>

                      {/* Meta Information */}
                      <div className="space-y-2 text-sm text-muted-foreground border-t pt-3">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">Resume:</span>
                          <span>{application.resume?.name || 'Unknown Resume'}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <Calendar className="h-3.5 w-3.5" />
                          <span>
                            Applied{' '}
                            {formatDistanceToNow(new Date(application.applied_at), {
                              addSuffix: true,
                            })}
                          </span>
                        </div>
                        {application.last_activity_at && (
                          <div className="flex items-center gap-2">
                            <Clock className="h-3.5 w-3.5" />
                            <span>
                              Updated{' '}
                              {formatDistanceToNow(new Date(application.last_activity_at), {
                                addSuffix: true,
                              })}
                            </span>
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              </div>
            ))}
          </div>

          {/* Pagination */}
          {pagination && pagination.total > pageSize && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                {t('common.previous')}
              </Button>
              <span className="flex items-center px-4 text-sm text-muted-foreground">
                Page {page + 1} of {Math.ceil(pagination.total / pageSize)}
              </span>
              <Button
                variant="outline"
                onClick={() => setPage((p) => p + 1)}
                disabled={(page + 1) * pageSize >= pagination.total}
              >
                {t('common.next')}
              </Button>
            </div>
          )}
        </>
      )}

      {/* Modals */}
      <CreateApplicationModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
      />

      {activeQuickAction?.type === 'comment' && (
        <AddCommentModal
          open={true}
          onOpenChange={(open) => !open && setActiveQuickAction(null)}
          applicationId={activeQuickAction.application.id}
        />
      )}

      {activeQuickAction?.type === 'stage' && (
        <AddStageModal
          open={true}
          onOpenChange={(open) => !open && setActiveQuickAction(null)}
          applicationId={activeQuickAction.application.id}
        />
      )}

      {activeQuickAction?.type === 'archive' && (
        <UpdateApplicationStatusModal
          open={true}
          onOpenChange={(open) => {
            if (!open) {
              setActiveQuickAction(null);
              refetch();
            }
          }}
          applicationId={activeQuickAction.application.id}
          currentStatus={activeQuickAction.application.status}
        />
      )}
    </div>
  );
}
