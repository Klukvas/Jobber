import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { resumesService } from '@/services/resumesService';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/shared/ui/Dialog';
import { Button } from '@/shared/ui/Button';
import { AlertTriangle } from 'lucide-react';
import type { ResumeDTO } from '@/shared/types/api';

interface DeleteResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resume: ResumeDTO;
}

export function DeleteResumeModal({ open, onOpenChange, resume }: DeleteResumeModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: () => resumesService.delete(resume.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resumes'] });
      onOpenChange(false);
    },
  });

  const handleDelete = () => {
    deleteMutation.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            <DialogTitle>Delete Resume</DialogTitle>
          </div>
          <DialogDescription>
            Are you sure you want to delete "{resume.title}"?
          </DialogDescription>
        </DialogHeader>
        
        <div className="py-4">
          {(resume.applications_count ?? 0) > 0 && (
            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
              <p className="font-medium">⚠️ This resume is currently used in {resume.applications_count} application{resume.applications_count === 1 ? '' : 's'}.</p>
              <p className="mt-1">You cannot delete it until it's removed from all applications.</p>
            </div>
          )}
          
          {(resume.applications_count ?? 0) === 0 && (
            <p className="text-sm text-muted-foreground">
              This action cannot be undone. The resume will be permanently deleted.
            </p>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            {t('common.cancel')}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={resume.can_delete === false || deleteMutation.isPending}
          >
            {deleteMutation.isPending ? t('common.loading') : 'Delete Resume'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
