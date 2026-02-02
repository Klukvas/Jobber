import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { commentsService } from '@/services/commentsService';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/shared/ui/Dialog';
import { Button } from '@/shared/ui/Button';
import { Label } from '@/shared/ui/Label';

interface AddCommentModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  stageId?: string;
  stageName?: string;
}

export function AddCommentModal({
  open,
  onOpenChange,
  applicationId,
  stageId,
  stageName,
}: AddCommentModalProps) {
  const queryClient = useQueryClient();
  const [content, setContent] = useState('');

  const createMutation = useMutation({
    mutationFn: commentsService.create,
    onSuccess: () => {
      // Invalidate application query to refresh embedded comments
      queryClient.invalidateQueries({ queryKey: ['application', applicationId] });
      onOpenChange(false);
      setContent('');
    },
    onError: (error) => {
      console.error('Failed to add comment:', error);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (content.trim()) {
      createMutation.mutate({
        application_id: applicationId,
        stage_id: stageId,
        content: content.trim(),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Add Comment</DialogTitle>
          <DialogDescription>
            {stageId && stageName
              ? `Add a comment for the "${stageName}" stage`
              : 'Add a general comment for this application'}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="content">Comment *</Label>
              <textarea
                id="content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Enter your comment here..."
                className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? 'Adding...' : 'Add Comment'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
