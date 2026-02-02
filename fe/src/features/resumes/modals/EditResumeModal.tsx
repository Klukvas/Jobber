import { useState, useEffect } from 'react';
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
import { Input } from '@/shared/ui/Input';
import { Label } from '@/shared/ui/Label';
import { Checkbox } from '@/shared/ui/Checkbox';
import { showSuccessNotification, showErrorNotification } from '@/shared/lib/notifications';
import type { ResumeDTO } from '@/shared/types/api';

interface EditResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resume: ResumeDTO;
}

export function EditResumeModal({ open, onOpenChange, resume }: EditResumeModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState(resume.title);
  const [fileUrl, setFileUrl] = useState(resume.file_url || '');
  const [isActive, setIsActive] = useState(resume.is_active);

  // Update form when resume changes
  useEffect(() => {
    setTitle(resume.title);
    setFileUrl(resume.file_url || '');
    setIsActive(resume.is_active);
  }, [resume]);

  const updateMutation = useMutation({
    mutationFn: (data: { title?: string; file_url?: string | null; is_active?: boolean }) =>
      resumesService.update(resume.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resumes'] });
      showSuccessNotification('Resume updated successfully');
      onOpenChange(false);
    },
    onError: (error: any) => {
      showErrorNotification(error?.message || 'Failed to update resume');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const updateData: { title: string; file_url?: string | null; is_active: boolean } = {
      title,
      is_active: isActive,
    };

    // Only include file_url for external resumes
    if (resume.storage_type === 'external') {
      updateData.file_url = fileUrl || null;
    }

    updateMutation.mutate(updateData);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Edit Resume</DialogTitle>
          <DialogDescription>Update your resume information</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-title">Title *</Label>
              <Input
                id="edit-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g., Software Engineer Resume - 2024"
                required
              />
            </div>

            {/* Storage Type Indicator */}
            <div className="space-y-2">
              <Label>Storage Type</Label>
              <div className="text-sm text-muted-foreground">
                {resume.storage_type === 's3' ? (
                  <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
                    <span className="font-medium">Cloud Storage (PDF)</span>
                    <span className="text-xs">• File cannot be changed</span>
                  </div>
                ) : (
                  <span className="px-3 py-2 bg-muted rounded-md inline-block">External URL</span>
                )}
              </div>
            </div>

            {/* File URL - Only editable for external resumes */}
            {resume.storage_type === 'external' && (
              <div className="space-y-2">
                <Label htmlFor="edit-fileUrl">File URL</Label>
                <Input
                  id="edit-fileUrl"
                  type="url"
                  value={fileUrl}
                  onChange={(e) => setFileUrl(e.target.value)}
                  placeholder="https://example.com/my-resume.pdf"
                />
                <p className="text-xs text-muted-foreground">
                  Leave empty to remove the external link
                </p>
              </div>
            )}

            <div className="flex items-center space-x-2">
              <Checkbox
                id="edit-isActive"
                checked={isActive}
                onCheckedChange={(checked) => setIsActive(checked as boolean)}
              />
              <Label htmlFor="edit-isActive" className="cursor-pointer">
                Mark as active resume
              </Label>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t('common.loading') : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
