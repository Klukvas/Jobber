import { useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Upload, FileText, Loader2, X } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/Dialog";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";

interface ImportResumeModalProps {
  readonly open: boolean;
  readonly onOpenChange: (open: boolean) => void;
}

export function ImportResumeModal({ open, onOpenChange }: ImportResumeModalProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [activeTab, setActiveTab] = useState<"text" | "pdf">("text");
  const [text, setText] = useState("");
  const [title, setTitle] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isImporting, setIsImporting] = useState(false);

  const resetForm = useCallback(() => {
    setText("");
    setTitle("");
    setSelectedFile(null);
    setIsImporting(false);
  }, []);

  const handleClose = useCallback(() => {
    if (!isImporting) {
      resetForm();
      onOpenChange(false);
    }
  }, [isImporting, resetForm, onOpenChange]);

  const handleImportText = useCallback(async () => {
    if (!text.trim()) return;
    setIsImporting(true);
    try {
      const result = await resumeBuilderService.importFromText({
        text: text.trim(),
        title: title.trim() || undefined,
      });
      showSuccessNotification(t("resumeBuilder.import.success"));
      onOpenChange(false);
      resetForm();
      navigate(`/app/resume-builder/${result.id}`);
    } catch (error) {
      showErrorNotification(
        error instanceof Error ? error.message : t("resumeBuilder.import.error")
      );
    } finally {
      setIsImporting(false);
    }
  }, [text, title, navigate, onOpenChange, resetForm, t]);

  const handleImportPDF = useCallback(async () => {
    if (!selectedFile) return;
    setIsImporting(true);
    try {
      const result = await resumeBuilderService.importFromPDF(
        selectedFile,
        title.trim() || undefined,
      );
      showSuccessNotification(t("resumeBuilder.import.success"));
      onOpenChange(false);
      resetForm();
      navigate(`/app/resume-builder/${result.id}`);
    } catch (error) {
      showErrorNotification(
        error instanceof Error ? error.message : t("resumeBuilder.import.error")
      );
    } finally {
      setIsImporting(false);
    }
  }, [selectedFile, title, navigate, onOpenChange, resetForm, t]);

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > 5 * 1024 * 1024) {
        showErrorNotification(t("resumeBuilder.import.fileTooLarge"));
        return;
      }
      if (file.type !== "application/pdf") {
        showErrorNotification(t("resumeBuilder.import.invalidFileType"));
        return;
      }
      setSelectedFile(file);
    }
  }, [t]);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("resumeBuilder.import.title")}</DialogTitle>
        </DialogHeader>

        {/* Tabs */}
        <div className="flex border-b">
          <button
            type="button"
            className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === "text"
                ? "border-b-2 border-primary text-primary"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("text")}
          >
            <FileText className="mr-2 inline-block h-4 w-4" />
            {t("resumeBuilder.import.pasteText")}
          </button>
          <button
            type="button"
            className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === "pdf"
                ? "border-b-2 border-primary text-primary"
                : "text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setActiveTab("pdf")}
          >
            <Upload className="mr-2 inline-block h-4 w-4" />
            {t("resumeBuilder.import.uploadPdf")}
          </button>
        </div>

        {/* Title input (shared) */}
        <div>
          <label className="mb-1 block text-sm font-medium" htmlFor="import-title">
            {t("resumeBuilder.import.resumeTitle")}
          </label>
          <input
            id="import-title"
            type="text"
            className="w-full rounded-md border px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            placeholder={t("resumeBuilder.import.titlePlaceholder")}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            disabled={isImporting}
          />
        </div>

        {/* Tab content */}
        {activeTab === "text" ? (
          <div>
            <label className="mb-1 block text-sm font-medium" htmlFor="import-text">
              {t("resumeBuilder.import.resumeText")}
            </label>
            <textarea
              id="import-text"
              className="h-48 w-full resize-none rounded-md border px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder={t("resumeBuilder.import.textPlaceholder")}
              value={text}
              onChange={(e) => setText(e.target.value)}
              disabled={isImporting}
            />
            <p className="mt-1 text-xs text-muted-foreground">
              {t("resumeBuilder.import.textHint")}
            </p>
          </div>
        ) : (
          <div>
            <div
              className="flex h-32 cursor-pointer flex-col items-center justify-center rounded-md border-2 border-dashed transition-colors hover:border-primary hover:bg-muted/50"
              onClick={() => fileInputRef.current?.click()}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") fileInputRef.current?.click();
              }}
              role="button"
              tabIndex={0}
            >
              {selectedFile ? (
                <div className="flex items-center gap-2">
                  <FileText className="h-5 w-5 text-primary" />
                  <span className="text-sm">{selectedFile.name}</span>
                  <button
                    type="button"
                    className="rounded-full p-0.5 hover:bg-muted"
                    onClick={(e) => {
                      e.stopPropagation();
                      setSelectedFile(null);
                    }}
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ) : (
                <>
                  <Upload className="mb-2 h-8 w-8 text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">
                    {t("resumeBuilder.import.dropOrClick")}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {t("resumeBuilder.import.maxSize")}
                  </p>
                </>
              )}
            </div>
            <input
              ref={fileInputRef}
              type="file"
              accept=".pdf"
              className="hidden"
              onChange={handleFileChange}
            />
          </div>
        )}

        {/* Import button */}
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={handleClose} disabled={isImporting}>
            {t("common.cancel")}
          </Button>
          <Button
            onClick={activeTab === "text" ? handleImportText : handleImportPDF}
            disabled={
              isImporting ||
              (activeTab === "text" ? !text.trim() : !selectedFile)
            }
          >
            {isImporting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t("resumeBuilder.import.importing")}
              </>
            ) : (
              t("resumeBuilder.import.importButton")
            )}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
