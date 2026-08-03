import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { commentsService } from "@/services/commentsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Label } from "@/shared/ui/Label";
import { Loader2 } from "lucide-react";

interface AddCommentModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  jobId: string;
  stageId?: string;
  stageName?: string;
}

export function AddCommentModal({
  open,
  onOpenChange,
  jobId,
  stageId,
  stageName,
}: AddCommentModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [content, setContent] = useState("");

  const createMutation = useMutation({
    mutationFn: commentsService.create,
    onSuccess: () => {
      // Invalidate job query to refresh embedded comments
      queryClient.invalidateQueries({ queryKey: ["job", jobId] });
      showSuccessNotification(t("jobs.commentAddedSuccess"));
      onOpenChange(false);
      setContent("");
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.commentAddedError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (content.trim()) {
      createMutation.mutate({
        job_id: jobId,
        stage_id: stageId,
        content: content.trim(),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("jobs.addComment")}</DialogTitle>
          <DialogDescription>
            {stageId && stageName
              ? t("jobs.addCommentForStage", { stageName })
              : t("jobs.addCommentGeneral")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="content">{`${t("jobs.commentLabel")} *`}</Label>
              <textarea
                id="content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder={t("jobs.commentPlaceholder")}
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
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t("jobs.adding")}
                </>
              ) : (
                t("jobs.addComment")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
