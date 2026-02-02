import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { applicationsService } from '@/services/applicationsService';
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
import type { ApplicationStageDTO } from '@/shared/types/api';

interface UpdateStageStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  stage: ApplicationStageDTO;
}

const STAGE_STATUSES = [
  { value: 'pending', label: 'Pending', description: 'Not yet started' },
  { value: 'active', label: 'Active', description: 'Currently in progress' },
  { value: 'completed', label: 'Completed', description: 'Successfully finished' },
  { value: 'skipped', label: 'Skipped', description: 'Stage was skipped' },
  { value: 'cancelled', label: 'Cancelled', description: 'Stage was cancelled' },
];

export function UpdateStageStatusModal({
  open,
  onOpenChange,
  applicationId,
  stage,
}: UpdateStageStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState(stage.status);

  const updateStatusMutation = useMutation({
    mutationFn: (status: string) =>
      applicationsService.updateStage(applicationId, stage.id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['application-stages', applicationId] });
      queryClient.invalidateQueries({ queryKey: ['application', applicationId] });
      onOpenChange(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newStatus !== stage.status) {
      updateStatusMutation.mutate(newStatus);
    } else {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Change Stage Status</DialogTitle>
          <DialogDescription>
            Update the status of "{stage.stage_name}"
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Current Status</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {stage.status.charAt(0).toUpperCase() + stage.status.slice(1)}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">New Status *</Label>
              <select
                id="status"
                value={newStatus}
                onChange={(e) => setNewStatus(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                {STAGE_STATUSES.map((status) => (
                  <option key={status.value} value={status.value}>
                    {status.label} - {status.description}
                  </option>
                ))}
              </select>
            </div>
            {updateStatusMutation.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                Failed to update status. Please try again.
              </div>
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
              type="submit"
              disabled={updateStatusMutation.isPending || newStatus === stage.status}
            >
              {updateStatusMutation.isPending ? t('common.loading') : 'Update Status'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
