import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { looksLikePdf, resumesService } from "@/services/resumesService";
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

/** Mirrors the server's MaxUploadedResumeBytes. */
const MAX_UPLOAD_BYTES = 10 * 1024 * 1024;

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

  // File upload mutation. The server verifies the stored object and activates
  // the resume in one step, so a spoofed file never leaves a half-created row.
  const uploadMutation = useMutation({
    mutationFn: (file: File) =>
      resumesService.uploadResume(
        file,
        setUploadProgress,
        title && title !== "Untitled Resume" ? title : undefined,
      ),
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

  // Fast feedback before a pointless round-trip. None of this is a security
  // boundary — the extension and the MIME type both come from the client, and
  // even the magic-byte read below runs here. The server re-checks the stored
  // object in resumes/{id}/finalize, and that check is the one that decides.
  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const input = e.target;
    const file = input.files?.[0];
    if (!file) return;

    const reject = (messageKey: string) => {
      showErrorNotification(t(messageKey));
      input.value = "";
      setSelectedFile(null);
      setUploadProgress(0);
    };

    if (file.type !== "application/pdf") {
      reject("resumes.onlyPdfAllowed");
      return;
    }
    if (file.size > MAX_UPLOAD_BYTES) {
      reject("resumes.fileSizeLimit");
      return;
    }
    // `looksLikePdf` reads the file, and a read can fail outright: a file that
    // was moved or unmounted between picking and reading rejects with a
    // NotReadableError. Unhandled, that escaped this handler as a rejected
    // promise, left the selection half-applied and told the customer nothing.
    // Unreadable and not-a-PDF land in the same place, because from here they
    // are the same thing: this file cannot be uploaded.
    let isPdf = false;
    try {
      isPdf = await looksLikePdf(file);
    } catch {
      isPdf = false;
    }
    if (!isPdf) {
      reject("resumes.notARealPdf");
      return;
    }

    setSelectedFile(file);
    // Auto-fill title from filename if empty
    if (!title) {
      const fileName = file.name.replace(/\.[^/.]+$/, ""); // Remove extension
      setTitle(fileName);
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
                  onChange={(e) => void handleFileChange(e)}
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
