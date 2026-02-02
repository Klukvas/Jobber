import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { applicationsService } from '@/services/applicationsService';
import { commentsService } from '@/services/commentsService';
import { Button } from '@/shared/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/Card';
import { SkeletonDetail } from '@/shared/ui/Skeleton';
import { ErrorState } from '@/shared/ui/ErrorState';
import { ArrowLeft, Calendar, Plus, MessageSquarePlus, Edit } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { Timeline } from '@/features/applications/components/Timeline';
import { AddStageModal } from '@/features/applications/modals/AddStageModal';
import { UpdateApplicationStatusModal } from '@/features/applications/modals/UpdateApplicationStatusModal';
import { Textarea } from '@/shared/ui/Textarea';

export default function ApplicationDetail() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isAddStageModalOpen, setIsAddStageModalOpen] = useState(false);
  const [isUpdateStatusModalOpen, setIsUpdateStatusModalOpen] = useState(false);
  const [newComment, setNewComment] = useState('');

  const { data: application, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['application', id],
    queryFn: () => applicationsService.getById(id!),
    enabled: !!id,
  });

  const { data: stages } = useQuery({
    queryKey: ['application-stages', id],
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
      queryClient.invalidateQueries({ queryKey: ['application', id] });
      setNewComment('');
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
        <Button variant="ghost" onClick={() => navigate('/app/applications')}>
          <ArrowLeft className="h-4 w-4" />
          {t('common.back')}
        </Button>
        <SkeletonDetail />
      </div>
    );
  }

  if (isError || !application) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" onClick={() => navigate('/app/applications')}>
          <ArrowLeft className="h-4 w-4" />
          {t('common.back')}
        </Button>
        <ErrorState
          message={error?.message || 'Application not found'}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate('/app/applications')}>
          <ArrowLeft className="h-4 w-4" />
          {t('common.back')}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('applications.details')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-sm text-muted-foreground">Application Name</p>
            <p className="font-semibold text-lg">
              {application.name || application.job_title || 'Untitled Application'}
            </p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Job</p>
            <p className="font-medium">{application.job_title || application.job_id}</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Resume</p>
            <p className="font-medium">{application.resume_title || application.resume_id}</p>
          </div>
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">
              Applied{' '}
              {formatDistanceToNow(new Date(application.applied_at), {
                addSuffix: true,
              })}
            </span>
          </div>
          <div>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <p
                  className={`font-medium ${
                    application.status === 'active'
                      ? 'text-green-600'
                      : 'text-muted-foreground'
                  }`}
                >
                  {application.status.charAt(0).toUpperCase() + application.status.slice(1)}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsUpdateStatusModalOpen(true)}
              >
                <Edit className="h-4 w-4 mr-2" />
                Change Status
              </Button>
            </div>
          </div>

          <div className="mt-6 border-t pt-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold">Comments</h3>
            </div>
            
            {applicationComments.length > 0 && (
              <div className="space-y-3 mb-4">
                {applicationComments.map((comment) => (
                  <div
                    key={comment.id}
                    className="rounded-lg border bg-muted/50 p-3"
                  >
                    <p className="text-sm whitespace-pre-wrap">{comment.content}</p>
                    <p className="text-xs text-muted-foreground mt-2">
                      {formatDistanceToNow(new Date(comment.created_at), {
                        addSuffix: true,
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
                placeholder="Add a comment..."
                className="flex-1"
                rows={3}
              />
              <Button
                type="submit"
                size="sm"
                disabled={!newComment.trim() || addCommentMutation.isPending}
              >
                <MessageSquarePlus className="h-4 w-4 mr-2" />
                Add Comment
              </Button>
            </form>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle>{t('applications.timeline')}</CardTitle>
          <Button
            size="sm"
            onClick={() => setIsAddStageModalOpen(true)}
          >
            <Plus className="h-4 w-4" />
            Add new stage
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
    </div>
  );
}
