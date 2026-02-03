import { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { resumesService } from '@/services/resumesService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonList } from '@/shared/ui/Skeleton';
import { EmptyState } from '@/shared/ui/EmptyState';
import { ErrorState } from '@/shared/ui/ErrorState';
import { 
  Plus, 
  FileText, 
  ExternalLink, 
  CheckCircle, 
  XCircle,
  Calendar,
  MoreVertical,
  Edit,
  Trash2,
  ArrowUpDown,
  Briefcase,
  Download,
  Cloud,
  Link as LinkIcon,
} from 'lucide-react';
import { format } from 'date-fns';
import { CreateResumeModal } from '@/features/resumes/modals/CreateResumeModal';
import { EditResumeModal } from '@/features/resumes/modals/EditResumeModal';
import { DeleteResumeModal } from '@/features/resumes/modals/DeleteResumeModal';
import { showErrorNotification } from '@/shared/lib/notifications';
import { usePageTitle } from '@/shared/lib/usePageTitle';
import type { ResumeDTO } from '@/shared/types/api';

type SortBy = 'created_at' | 'title' | 'is_active';
type SortDir = 'asc' | 'desc';

export default function Resumes() {
  const { t } = useTranslation();
  usePageTitle('resumes.title');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [sortBy, setSortBy] = useState<SortBy>('created_at');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [editingResume, setEditingResume] = useState<ResumeDTO | null>(null);
  const [deletingResume, setDeletingResume] = useState<ResumeDTO | null>(null);

  // Close context menu when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setOpenMenuId(null);
    if (openMenuId) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['resumes', sortBy, sortDir],
    queryFn: () => resumesService.list({ 
      limit: 100, 
      offset: 0,
      sort_by: sortBy,
      sort_dir: sortDir,
    }),
  });

  const toggleSort = (field: SortBy) => {
    if (sortBy === field) {
      setSortDir(sortDir === 'desc' ? 'asc' : 'desc');
    } else {
      setSortBy(field);
      setSortDir('desc');
    }
  };

  const handleEdit = (resume: ResumeDTO) => {
    setEditingResume(resume);
    setOpenMenuId(null);
  };

  const handleDelete = (resume: ResumeDTO) => {
    setDeletingResume(resume);
    setOpenMenuId(null);
  };

  // Handle S3 resume download
  const downloadMutation = useMutation({
    mutationFn: resumesService.generateDownloadURL,
    onSuccess: (data) => {
      // Open download URL in new tab
      window.open(data.download_url, '_blank');
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || 'Failed to generate download link');
    },
  });

  const handleDownload = (resumeId: string) => {
    downloadMutation.mutate(resumeId);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t('resumes.title')}</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t('resumes.title')}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const resumes = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t('resumes.title')}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t('resumes.create')}
        </Button>
      </div>

      {resumes.length === 0 ? (
        <EmptyState
          icon={<FileText className="h-12 w-12" />}
          title="No resumes yet"
          description="Create your first resume to start applying for jobs. You can upload multiple versions and track which ones you use for different applications."
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t('resumes.create')}
            </Button>
          }
        />
      ) : (
        <>
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">Sort by:</span>
            <Button
              variant={sortBy === 'created_at' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('created_at')}
            >
              <Calendar className="h-3 w-3 mr-1" />
              Created Date
              {sortBy === 'created_at' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortBy === 'title' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('title')}
            >
              <FileText className="h-3 w-3 mr-1" />
              Title
              {sortBy === 'title' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
            <Button
              variant={sortBy === 'is_active' ? 'default' : 'outline'}
              size="sm"
              onClick={() => toggleSort('is_active')}
            >
              Active Status
              {sortBy === 'is_active' && (
                <ArrowUpDown className="h-3 w-3 ml-1" />
              )}
            </Button>
          </div>

          {/* Resume Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {resumes.map((resume) => (
              <Card 
                key={resume.id} 
                className={`transition-all hover:shadow-md h-full group relative ${
                  !resume.is_active ? 'opacity-60' : ''
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg font-bold leading-tight flex-1">
                      {resume.title}
                    </CardTitle>
                    {/* Context Menu */}
                    <div className="relative" onClick={(e) => e.preventDefault()}>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          e.preventDefault();
                          setOpenMenuId(openMenuId === resume.id ? null : resume.id);
                        }}
                        className="p-1 rounded-md hover:bg-accent transition-colors opacity-0 group-hover:opacity-100"
                        aria-label="Resume actions"
                      >
                        <MoreVertical className="h-4 w-4" />
                      </button>
                      {openMenuId === resume.id && (
                        <div className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10">
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              handleEdit(resume);
                            }}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <Edit className="h-4 w-4" />
                            Edit
                          </button>
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              handleDelete(resume);
                            }}
                            disabled={resume.can_delete === false}
                            className={`flex items-center gap-2 w-full px-3 py-2 text-sm text-left ${
                              resume.can_delete !== false
                                ? 'hover:bg-accent text-destructive' 
                                : 'opacity-50 cursor-not-allowed'
                            }`}
                            title={resume.can_delete === false ? 'Cannot delete resume used in applications' : ''}
                          >
                            <Trash2 className="h-4 w-4" />
                            Delete
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                  
                  {/* Active/Inactive Badge */}
                  <div className="flex items-center gap-2 mt-2">
                    {resume.is_active ? (
                      <div className="flex items-center gap-1 text-xs font-medium text-green-600 bg-green-50 px-2 py-1 rounded">
                        <CheckCircle className="h-3 w-3" />
                        Active
                      </div>
                    ) : (
                      <div className="flex items-center gap-1 text-xs font-medium text-muted-foreground bg-muted px-2 py-1 rounded">
                        <XCircle className="h-3 w-3" />
                        Inactive
                      </div>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* File Access Section */}
                  <div className="space-y-2">
                    {resume.storage_type === 's3' ? (
                      <>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <Cloud className="h-3 w-3" />
                          <span>Cloud Storage (PDF)</span>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDownload(resume.id)}
                          disabled={downloadMutation.isPending}
                          className="w-full"
                        >
                          <Download className="h-3 w-3 mr-2" />
                          {downloadMutation.isPending ? 'Generating link...' : 'Download Resume'}
                        </Button>
                      </>
                    ) : resume.file_url ? (
                      <>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <LinkIcon className="h-3 w-3" />
                          <span>External URL</span>
                        </div>
                        <a
                          href={resume.file_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center justify-center gap-2 text-sm hover:underline w-full px-4 py-2 border rounded-md hover:bg-accent transition-colors"
                        >
                          <ExternalLink className="h-3 w-3" />
                          View Resume
                        </a>
                      </>
                    ) : (
                      <div className="text-sm text-muted-foreground italic">
                        No file attached
                      </div>
                    )}
                  </div>
                  
                  {/* Usage Indicator */}
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Briefcase className="h-4 w-4" />
                    <span>
                      {(resume.applications_count ?? 0) === 0 
                        ? 'Not used in applications yet' 
                        : `Used in ${resume.applications_count} application${resume.applications_count === 1 ? '' : 's'}`
                      }
                    </span>
                  </div>
                  
                  <div className="text-sm text-muted-foreground">
                    Created {format(new Date(resume.created_at), 'MMM d, yyyy')}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      <CreateResumeModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
      />

      {editingResume && (
        <EditResumeModal
          open={!!editingResume}
          onOpenChange={(open) => !open && setEditingResume(null)}
          resume={editingResume}
        />
      )}

      {deletingResume && (
        <DeleteResumeModal
          open={!!deletingResume}
          onOpenChange={(open) => !open && setDeletingResume(null)}
          resume={deletingResume}
        />
      )}
    </div>
  );
}
