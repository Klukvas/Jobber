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
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [content, setContent] = useState("");

  const createMutation = useMutation({
    mutationFn: commentsService.create,
    onSuccess: () => {
      // Invalidate application query to refresh embedded comments
      queryClient.invalidateQueries({
        queryKey: ["application", applicationId],
      });
      showSuccessNotification(t("applications.commentAddedSuccess"));
      onOpenChange(false);
      setContent("");
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("applications.commentAddedError"),
      );
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
          <DialogTitle>{t("applications.addComment")}</DialogTitle>
          <DialogDescription>
            {stageId && stageName
              ? t("applications.addCommentForStage", { stageName })
              : t("applications.addCommentGeneral")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="content">{`${t("applications.commentLabel")} *`}</Label>
              <textarea
                id="content"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder={t("applications.commentPlaceholder")}
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
              {createMutation.isPending
                ? t("applications.adding")
                : t("applications.addComment")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
