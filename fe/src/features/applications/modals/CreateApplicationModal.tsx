import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { applicationsService } from '@/services/applicationsService';
import { resumesService } from '@/services/resumesService';
import { jobsService } from '@/services/jobsService';
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

interface CreateApplicationModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateApplicationModal({
  open,
  onOpenChange,
}: CreateApplicationModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [jobId, setJobId] = useState('');
  const [resumeId, setResumeId] = useState('');

  const { data: jobsData } = useQuery({
    queryKey: ['jobs'],
    queryFn: () => jobsService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const { data: resumesData } = useQuery({
    queryKey: ['resumes'],
    queryFn: () => resumesService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: applicationsService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications'] });
      onOpenChange(false);
      setName('');
      setJobId('');
      setResumeId('');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name && jobId && resumeId) {
      createMutation.mutate({
        name: name,
        job_id: jobId,
        resume_id: resumeId,
        applied_at: new Date().toISOString(),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t('applications.create')}</DialogTitle>
          <DialogDescription>
            Create a new job application by selecting a job and resume
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Application Name *</Label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Backend Developer @ Acme Corp"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="job">Job *</Label>
              <select
                id="job"
                value={jobId}
                onChange={(e) => setJobId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                <option value="">Select a job</option>
                {jobsData?.items?.map((job) => (
                  <option key={job.id} value={job.id}>
                    {job.title}{job.company_name ? ` (${job.company_name})` : ''}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="resume">Resume *</Label>
              <select
                id="resume"
                value={resumeId}
                onChange={(e) => setResumeId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                <option value="">Select a resume</option>
                {resumesData?.items?.map((resume) => (
                  <option key={resume.id} value={resume.id}>
                    {resume.title}
                  </option>
                ))}
              </select>
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
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? t('common.loading') : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
