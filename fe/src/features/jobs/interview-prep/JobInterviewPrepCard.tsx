import { useState } from "react";
import { useTranslation } from "react-i18next";
import { BookOpen } from "lucide-react";
import type { JobDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import { Card, CardHeader, CardTitle, CardContent } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { InterviewPrepDialog } from "./InterviewPrepDialog";

export type InterviewProps = {
  job: JobDTO;
  builderResumes: readonly ResumeBuilderDTO[];
  isLoading: boolean;
  isError: boolean;
  isJobDirty: boolean;
  isJobSaving: boolean;
};
export function JobInterviewPrepCard(props: InterviewProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{t("jobs.interview.title")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          {t("jobs.interview.intro")}
        </p>
        <Button variant="outline" onClick={() => setOpen(true)}>
          <BookOpen className="h-4 w-4" aria-hidden />
          {t("jobs.interview.open")}
        </Button>
        {open && (
          <InterviewPrepDialog {...props} onClose={() => setOpen(false)} />
        )}
      </CardContent>
    </Card>
  );
}
