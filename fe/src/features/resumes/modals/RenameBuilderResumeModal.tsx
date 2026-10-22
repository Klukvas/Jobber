import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { renameResumeDocument } from "@/features/resume-builder/lib/resumeDocumentSync";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
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
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { Loader2 } from "lucide-react";
import type { FullResumeDTO, ResumeBuilderDTO } from "@/shared/types/resume-builder";

interface RenameBuilderResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resume: ResumeBuilderDTO;
}

function ModalContent({
  resume,
  onOpenChange,
}: Omit<RenameBuilderResumeModalProps, "open">) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [title, setTitle] = useState(resume.title);

  const renameMutation = useMutation({
    mutationFn: async (newTitle: string) => {
      await queryClient.cancelQueries({ queryKey: resumeQueryKeys.builderDetail(resume.id) });
      return renameResumeDocument(resume.id, newTitle);
    },
    onSuccess: (updated, newTitle) => {
      queryClient.setQueryData<FullResumeDTO>(resumeQueryKeys.builderDetail(resume.id), cached =>
        cached ? { ...cached, title: newTitle, updated_at: updated.updated_at } : cached,
      );
      void queryClient.invalidateQueries({
        queryKey: resumeQueryKeys.builderDetail(resume.id),
        refetchType: "none",
      });
      queryClient.invalidateQueries({
        queryKey: resumeQueryKeys.builderList(),
      });
      showSuccessNotification(t("resumes.updateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || t("resumes.updateError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = title.trim();
    if (!trimmed) return;
    renameMutation.mutate(trimmed);
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("common.rename")}</DialogTitle>
        <DialogDescription>
          {t("resumes.editDescription")}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <div className="py-4">
          <div className="space-y-2">
            <Label htmlFor="rename-title">
              {t("resumes.titleLabel")} *
            </Label>
            <Input
              id="rename-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("resumes.titlePlaceholder")}
              required
              autoFocus
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
          <Button type="submit" disabled={renameMutation.isPending}>
            {renameMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : (
              t("common.save")
            )}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function RenameBuilderResumeModal({
  open,
  onOpenChange,
  resume,
}: RenameBuilderResumeModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={resume.id}
          resume={resume}
          onOpenChange={onOpenChange}
        />
      </DialogContent>
    </Dialog>
  );
}
