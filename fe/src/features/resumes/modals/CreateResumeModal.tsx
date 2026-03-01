import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { resumesService } from "@/services/resumesService";
import type { ResumeDTO } from "@/shared/types/api";
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
  showErrorNotification,
  showSuccessNotification,
} from "@/shared/lib/notifications";
import { Loader2 } from "lucide-react";

interface CreateResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (resume: ResumeDTO) => void;
}

type UploadMode = "url" | "file";

export function CreateResumeModal({
  open,
  onOpenChange,
  onCreated,
}: CreateResumeModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [mode, setMode] = useState<UploadMode>("url");
  const [title, setTitle] = useState("");
  const [fileUrl, setFileUrl] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);

  // Traditional URL-based resume creation
  const createMutation = useMutation({
    mutationFn: resumesService.create,
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["resumes"] });
      showSuccessNotification(t("resumes.createSuccess"));
      onCreated?.(data);
      resetAndClose();
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || t("resumes.createError"));
    },
  });

  // File upload mutation
  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const resume = await resumesService.uploadResume(file, setUploadProgress);
      // Update title if provided
      if (title && title !== "Untitled Resume") {
        return resumesService.update(resume.id, { title, is_active: true });
      }
      return resumesService.update(resume.id, { is_active: true });
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["resumes"] });
      showSuccessNotification(t("resumes.uploadSuccess"));
      onCreated?.(data);
      resetAndClose();
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || t("resumes.uploadError"));
      setUploadProgress(0);
    },
  });

  const resetAndClose = () => {
    onOpenChange(false);
    setTimeout(() => {
      setTitle("");
      setFileUrl("");
      setSelectedFile(null);
      setUploadProgress(0);
      setMode("url");
    }, 300);
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      // Validate file type
      if (file.type !== "application/pdf") {
        showErrorNotification(t("resumes.onlyPdfAllowed"));
        e.target.value = "";
        return;
      }
      // Validate file size (max 10MB)
      if (file.size > 10 * 1024 * 1024) {
        showErrorNotification(t("resumes.fileSizeLimit"));
        e.target.value = "";
        return;
      }
      setSelectedFile(file);
      // Auto-fill title from filename if empty
      if (!title) {
        const fileName = file.name.replace(/\.[^/.]+$/, ""); // Remove extension
        setTitle(fileName);
      }
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (mode === "url") {
      if (title && fileUrl) {
        createMutation.mutate({ title, file_url: fileUrl, is_active: true });
      }
    } else {
      if (selectedFile) {
        uploadMutation.mutate(selectedFile);
      }
    }
  };

  const isLoading = createMutation.isPending || uploadMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={resetAndClose}>
        <DialogHeader>
          <DialogTitle>{t("resumes.create")}</DialogTitle>
          <DialogDescription>
            {t("resumes.createDescription")}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {/* Mode Selection - Switch Toggle */}
            <div className="space-y-2">
              <Label>{t("resumes.uploadMethod")}</Label>
              <div className="flex items-center gap-3 p-1 bg-muted rounded-lg w-fit">
                <button
                  type="button"
                  onClick={() => setMode("url")}
                  disabled={isLoading}
                  className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
                    mode === "url"
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {t("resumes.externalUrlOption")}
                </button>
                <button
                  type="button"
                  onClick={() => setMode("file")}
                  disabled={isLoading}
                  className={`px-4 py-2 text-sm font-medium rounded-md transition-all ${
                    mode === "file"
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {t("resumes.uploadPdfOption")}
                </button>
              </div>
            </div>

            {/* Title Field */}
            <div className="space-y-2">
              <Label htmlFor="title">
                {t("resumes.titleLabel")} {mode === "url" ? "*" : ""}
              </Label>
              <Input
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={t("resumes.titlePlaceholder")}
                required={mode === "url"}
                disabled={isLoading}
              />
              {mode === "file" && (
                <p className="text-xs text-muted-foreground">
                  {t("resumes.titleAutoFillHint")}
                </p>
              )}
            </div>

            {/* URL Mode */}
            {mode === "url" && (
              <div className="space-y-2">
                <Label htmlFor="fileUrl">{t("resumes.fileUrlLabel")}</Label>
                <Input
                  id="fileUrl"
                  type="url"
                  value={fileUrl}
                  onChange={(e) => setFileUrl(e.target.value)}
                  placeholder={t("resumes.fileUrlPlaceholder")}
                  required
                  disabled={isLoading}
                />
                <p className="text-xs text-muted-foreground">
                  {t("resumes.fileUrlHint")}
                </p>
              </div>
            )}

            {/* File Upload Mode */}
            {mode === "file" && (
              <div className="space-y-2">
                <Label htmlFor="file">{t("resumes.pdfFileLabel")}</Label>
                <Input
                  id="file"
                  type="file"
                  accept="application/pdf,.pdf"
                  onChange={handleFileChange}
                  required
                  disabled={isLoading}
                  className="cursor-pointer"
                />
                {selectedFile && (
                  <div className="text-sm text-muted-foreground">
                    <p>
                      {t("resumes.selectedFile", { name: selectedFile.name })}
                    </p>
                    <p className="text-xs">
                      {t("resumes.fileSize", {
                        size: (selectedFile.size / 1024).toFixed(2),
                      })}
                    </p>
                  </div>
                )}
                <p className="text-xs text-muted-foreground">
                  {t("resumes.pdfOnlyHint")}
                </p>
              </div>
            )}

            {/* Upload Progress */}
            {uploadMutation.isPending && uploadProgress > 0 && (
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span>{t("resumes.uploading")}</span>
                  <span>{uploadProgress}%</span>
                </div>
                <div className="w-full bg-muted rounded-full h-2">
                  <div
                    className="bg-primary h-2 rounded-full transition-all duration-300"
                    style={{ width: `${uploadProgress}%` }}
                  />
                </div>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={resetAndClose}
              disabled={isLoading}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="submit"
              disabled={
                isLoading ||
                (mode === "url" && (!title || !fileUrl)) ||
                (mode === "file" && !selectedFile)
              }
            >
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {uploadMutation.isPending
                    ? t("resumes.uploading")
                    : t("common.loading")}
                </>
              ) : mode === "file" ? (
                t("resumes.upload")
              ) : (
                t("common.create")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
