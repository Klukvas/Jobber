import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Loader2 } from "lucide-react";
import { applicationsService } from "@/services/applicationsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { resumesService } from "@/services/resumesService";
import { jobsService } from "@/services/jobsService";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { CreateResumeModal } from "@/features/resumes/modals/CreateResumeModal";
import type { JobDTO, ResumeDTO } from "@/shared/types/api";

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
  const [name, setName] = useState("");
  const [jobId, setJobId] = useState("");
  const [resumeId, setResumeId] = useState("");
  const [isCreateJobOpen, setIsCreateJobOpen] = useState(false);
  const [isCreateResumeOpen, setIsCreateResumeOpen] = useState(false);

  const { data: jobsData } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => jobsService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const { data: resumesData } = useQuery({
    queryKey: ["resumes"],
    queryFn: () => resumesService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const createMutation = useMutation({
    mutationFn: applicationsService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["applications"] });
      showSuccessNotification(t("applications.createSuccess"));
      onOpenChange(false);
      setName("");
      setJobId("");
      setResumeId("");
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("applications.createError"));
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

  const handleJobCreated = (job: JobDTO) => {
    setJobId(job.id);
  };

  const handleResumeCreated = (resume: ResumeDTO) => {
    setResumeId(resume.id);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent onClose={() => onOpenChange(false)}>
          <DialogHeader>
            <DialogTitle>{t("applications.create")}</DialogTitle>
            <DialogDescription>
              {t("applications.createDescription")}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit}>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">{`${t("applications.applicationName")} *`}</Label>
                <Input
                  id="name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t("applications.applicationNamePlaceholder")}
                  required
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="job">{`${t("applications.job")} *`}</Label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-auto py-0.5 px-2 text-xs"
                    onClick={() => setIsCreateJobOpen(true)}
                  >
                    <Plus className="h-3 w-3 mr-1" />
                    {t("applications.quickAddJob")}
                  </Button>
                </div>
                <select
                  id="job"
                  value={jobId}
                  onChange={(e) => setJobId(e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  required
                >
                  <option value="">{t("applications.selectJob")}</option>
                  {jobsData?.items?.map((job) => (
                    <option key={job.id} value={job.id}>
                      {job.title}
                      {job.company_name ? ` (${job.company_name})` : ""}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="resume">{`${t("applications.resumeLabel")} *`}</Label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-auto py-0.5 px-2 text-xs"
                    onClick={() => setIsCreateResumeOpen(true)}
                  >
                    <Plus className="h-3 w-3 mr-1" />
                    {t("applications.quickAddResume")}
                  </Button>
                </div>
                <select
                  id="resume"
                  value={resumeId}
                  onChange={(e) => setResumeId(e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  required
                >
                  <option value="">{t("applications.selectResume")}</option>
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
                {t("common.cancel")}
              </Button>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    {t("common.loading")}
                  </>
                ) : (
                  t("common.create")
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <CreateJobModal
        open={isCreateJobOpen}
        onOpenChange={setIsCreateJobOpen}
        onCreated={handleJobCreated}
      />
      <CreateResumeModal
        open={isCreateResumeOpen}
        onOpenChange={setIsCreateResumeOpen}
        onCreated={handleResumeCreated}
      />
    </>
  );
}
