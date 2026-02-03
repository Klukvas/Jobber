import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { jobsService } from '@/services/jobsService';
import { companiesService } from '@/services/companiesService';
import type { JobDTO, UpdateJobRequest } from '@/shared/types/api';
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

interface CreateJobModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  job?: JobDTO; // If provided, the modal is in edit mode
}

// Inner content component that resets state when key changes
function ModalContent({ job, onOpenChange, open }: CreateJobModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isEditMode = !!job;
  
  const [title, setTitle] = useState(job?.title || '');
  const [companyId, setCompanyId] = useState(job?.company_id || '');
  const [url, setUrl] = useState(job?.url || '');
  const [source, setSource] = useState(job?.source || '');
  const [notes, setNotes] = useState(job?.notes || '');

  const { data: companiesData } = useQuery({
    queryKey: ['companies'],
    queryFn: () => companiesService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: jobsService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      onOpenChange(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateJobRequest }) => 
      jobsService.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] });
      onOpenChange(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (title) {
      const data = {
        title,
        company_id: companyId || undefined,
        url: url || undefined,
        source: source || undefined,
        notes: notes || undefined,
      };

      if (isEditMode && job) {
        updateMutation.mutate({ id: job.id, data });
      } else {
        createMutation.mutate(data);
      }
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <>
      <DialogHeader>
        <DialogTitle>
          {isEditMode ? t('jobs.edit') : t('jobs.create')}
        </DialogTitle>
        <DialogDescription>
          {isEditMode 
            ? 'Update the job posting details' 
            : 'Add a new job posting to track'}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="title">{t('jobs.title_field')} *</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g., Senior Software Engineer"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="company">{t('jobs.company')}</Label>
            <select
              id="company"
              value={companyId}
              onChange={(e) => setCompanyId(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">Select a company (optional)</option>
              {(companiesData?.items || []).map((company) => (
                <option key={company.id} value={company.id}>
                  {company.name}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="url">{t('jobs.url')}</Label>
            <Input
              id="url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/jobs/123"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="source">{t('jobs.source')}</Label>
            <Input
              id="source"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder="e.g., LinkedIn, Indeed, Company Website"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="notes">{t('jobs.notes')}</Label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Any additional notes about the job..."
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            />
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
          <Button type="submit" disabled={isPending || !title}>
            {isPending 
              ? t('common.loading') 
              : isEditMode 
                ? t('common.save') 
                : t('common.create')}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function CreateJobModal({ open, onOpenChange, job }: CreateJobModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        {/* Key prop resets the form state when job changes or modal reopens */}
        <ModalContent 
          key={`${job?.id || 'new'}-${open}`}
          job={job} 
          onOpenChange={onOpenChange}
          open={open}
        />
      </DialogContent>
    </Dialog>
  );
}
