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

interface UpdateApplicationStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  currentStatus: string;
}

const APPLICATION_STATUSES = [
  { value: 'active', label: 'Active', description: 'Application is active' },
  { value: 'closed', label: 'Closed', description: 'Application is closed' },
];

export function UpdateApplicationStatusModal({
  open,
  onOpenChange,
  applicationId,
  currentStatus,
}: UpdateApplicationStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState(currentStatus);

  const updateStatusMutation = useMutation({
    mutationFn: (status: string) =>
      applicationsService.update(applicationId, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['application', applicationId] });
      queryClient.invalidateQueries({ queryKey: ['applications'] });
      onOpenChange(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newStatus !== currentStatus) {
      updateStatusMutation.mutate(newStatus);
    } else {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>Change Application Status</DialogTitle>
          <DialogDescription>
            Update the status of this application
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Current Status</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {currentStatus.charAt(0).toUpperCase() + currentStatus.slice(1)}
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
                {APPLICATION_STATUSES.map((status) => (
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
              disabled={updateStatusMutation.isPending || newStatus === currentStatus}
            >
              {updateStatusMutation.isPending ? t('common.loading') : 'Update Status'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
