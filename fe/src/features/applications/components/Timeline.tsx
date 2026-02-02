import { useState, useRef, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { ApplicationStageDTO, CommentDTO } from '@/shared/types/api';
import { applicationsService } from '@/services/applicationsService';
import { CheckCircle, Circle, Clock, Check, MoreVertical, Trash2, Edit, MessageSquare, Plus } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { Button } from '@/shared/ui/Button';
import { UpdateStageStatusModal } from '../modals/UpdateStageStatusModal';
import { AddCommentModal } from '../modals/AddCommentModal';

interface TimelineProps {
  stages: ApplicationStageDTO[];
  applicationId: string;
  stageComments: CommentDTO[];
}

export function Timeline({ stages, applicationId, stageComments }: TimelineProps) {
  const queryClient = useQueryClient();
  const [selectedStage, setSelectedStage] = useState<ApplicationStageDTO | null>(null);
  const [menuOpen, setMenuOpen] = useState<string | null>(null);
  const [confirmDeleteStage, setConfirmDeleteStage] = useState<ApplicationStageDTO | null>(null);
  const [commentModalOpen, setCommentModalOpen] = useState(false);
  const [commentStage, setCommentStage] = useState<{ id: string; name: string } | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const completeStage = useMutation({
    mutationFn: (stageId: string) =>
      applicationsService.completeStage(applicationId, stageId, {
        completed_at: new Date().toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['application-stages', applicationId] });
      queryClient.invalidateQueries({ queryKey: ['application', applicationId] });
    },
  });

  const deleteStage = useMutation({
    mutationFn: (stageId: string) =>
      applicationsService.deleteStage(applicationId, stageId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['application-stages', applicationId] });
      queryClient.invalidateQueries({ queryKey: ['application', applicationId] });
      setConfirmDeleteStage(null);
    },
    onError: (error) => {
      console.error('Failed to delete stage:', error);
    },
  });

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  const handleChangeStatus = (stage: ApplicationStageDTO) => {
    setSelectedStage(stage);
    setMenuOpen(null);
  };

  const handleDeleteClick = (stage: ApplicationStageDTO) => {
    setConfirmDeleteStage(stage);
    setMenuOpen(null);
  };

  const handleConfirmDelete = () => {
    if (confirmDeleteStage) {
      deleteStage.mutate(confirmDeleteStage.id);
    }
  };

  const handleAddComment = (stageId?: string, stageName?: string) => {
    if (stageId && stageName) {
      setCommentStage({ id: stageId, name: stageName });
    } else {
      setCommentStage(null);
    }
    setCommentModalOpen(true);
  };

  // Merge stages and stage comments into a single timeline
  const timelineItems = [
    ...stages.map(stage => ({ type: 'stage' as const, data: stage, timestamp: stage.started_at })),
    ...stageComments.map(comment => ({ type: 'comment' as const, data: comment, timestamp: comment.created_at }))
  ].sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
  return (
    <div className="relative space-y-6">
      {timelineItems.length === 0 && (
        <p className="text-center text-sm text-muted-foreground">
          No stages yet. Click "Add new stage" to get started.
        </p>
      )}

      {timelineItems.map((item, index) => {
        if (item.type === 'comment') {
          const comment = item.data as CommentDTO;
          const isLast = index === timelineItems.length - 1;
          const relatedStage = comment.stage_id 
            ? stages.find(s => s.id === comment.stage_id)
            : null;

          return (
            <div key={`comment-${comment.id}`} className="relative flex gap-4">
              {!isLast && (
                <div className="absolute left-[11px] top-8 h-full w-0.5 bg-border" />
              )}

              <div className="relative flex-shrink-0">
                <MessageSquare className="h-6 w-6 text-blue-500" />
              </div>

              <div className="flex-1 pb-6">
                <div className="rounded-lg border bg-muted/50 p-4">
                  {relatedStage && (
                    <p className="text-xs text-muted-foreground mb-2">
                      Comment on: {relatedStage.stage_name}
                    </p>
                  )}
                  <p className="text-sm whitespace-pre-wrap">{comment.content}</p>
                  <p className="text-xs text-muted-foreground mt-2">
                    {formatDistanceToNow(new Date(comment.created_at), {
                      addSuffix: true,
                    })}
                  </p>
                </div>
              </div>
            </div>
          );
        }

        const stage = item.data as ApplicationStageDTO;
        const isCompleted = stage.status === 'completed';
        const isActive = stage.status === 'active';
        const isLast = index === timelineItems.length - 1;

        return (
          <div key={stage.id} className="relative flex gap-4">
            {/* Timeline line */}
            {!isLast && (
              <div className="absolute left-[11px] top-8 h-full w-0.5 bg-border" />
            )}

            {/* Icon */}
            <div className="relative flex-shrink-0">
              {isCompleted ? (
                <CheckCircle className="h-6 w-6 text-green-600" />
              ) : isActive ? (
                <Clock className="h-6 w-6 text-blue-600" />
              ) : (
                <Circle className="h-6 w-6 text-muted-foreground" />
              )}
            </div>

            {/* Content */}
            <div className="flex-1 pb-6">
              <div className="flex items-center justify-between">
                <h4 className="font-semibold">{stage.stage_name}</h4>
                <div className="flex gap-2">
                  {isActive && !isCompleted && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => completeStage.mutate(stage.id)}
                      disabled={completeStage.isPending}
                    >
                      <Check className="h-4 w-4" />
                      Complete
                    </Button>
                  )}
                  <div className="relative">
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setMenuOpen(menuOpen === stage.id ? null : stage.id)}
                      title="Stage options"
                    >
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                    {menuOpen === stage.id && (
                      <div
                        ref={menuRef}
                        className="absolute right-0 z-10 mt-2 w-48 rounded-md border bg-popover shadow-lg"
                      >
                        <div className="py-1">
                          <button
                            onClick={() => handleChangeStatus(stage)}
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                          >
                            <Edit className="h-4 w-4" />
                            Change status
                          </button>
                          <button
                            onClick={() => handleAddComment(stage.id, stage.stage_name)}
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm hover:bg-accent"
                          >
                            <MessageSquare className="h-4 w-4" />
                            Add comment
                          </button>
                          <button
                            onClick={() => handleDeleteClick(stage)}
                            className="flex w-full items-center gap-2 px-4 py-2 text-sm text-destructive hover:bg-accent"
                          >
                            <Trash2 className="h-4 w-4" />
                            Delete
                          </button>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
              <p className="text-sm text-muted-foreground">
                Started{' '}
                {formatDistanceToNow(new Date(stage.started_at), {
                  addSuffix: true,
                })}
              </p>
              {stage.completed_at && (
                <p className="text-sm text-muted-foreground">
                  Completed{' '}
                  {formatDistanceToNow(new Date(stage.completed_at), {
                    addSuffix: true,
                  })}
                </p>
              )}
              <span
                className={`mt-2 inline-block rounded-full px-2 py-1 text-xs font-medium ${
                  isCompleted
                    ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100'
                    : isActive
                    ? 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-100'
                    : 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100'
                }`}
              >
                {stage.status}
              </span>
            </div>
          </div>
        );
      })}

      <AddCommentModal
        open={commentModalOpen}
        onOpenChange={setCommentModalOpen}
        applicationId={applicationId}
        stageId={commentStage?.id}
        stageName={commentStage?.name}
      />

      {selectedStage && (
        <UpdateStageStatusModal
          open={!!selectedStage}
          onOpenChange={(open) => !open && setSelectedStage(null)}
          applicationId={applicationId}
          stage={selectedStage}
        />
      )}

      {/* Delete confirmation modal */}
      {confirmDeleteStage && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
            <h2 className="text-xl font-semibold mb-4">Delete Stage</h2>
            <p className="text-sm text-muted-foreground mb-6">
              Are you sure you want to delete the stage "{confirmDeleteStage.stage_name}"? 
              This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setConfirmDeleteStage(null)}
                disabled={deleteStage.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleConfirmDelete}
                disabled={deleteStage.isPending}
              >
                {deleteStage.isPending ? 'Deleting...' : 'Delete'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
