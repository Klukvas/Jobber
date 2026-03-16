import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Download,
  Sparkles,
  ShieldCheck,
  BookOpen,
  Loader2,
  Undo2,
  Redo2,
  ChevronDown,
  FileText,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Tooltip } from "@/shared/ui/Tooltip";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useExportPDF } from "../hooks/useExportPDF";
import { useExportDOCX } from "../hooks/useExportDOCX";
import { useUndoRedo } from "../hooks/useUndoRedo";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";

interface EditorToolbarProps {
  readonly showAI: boolean;
  readonly onToggleAI: () => void;
  readonly showATS: boolean;
  readonly onToggleATS: () => void;
  readonly showContentLibrary: boolean;
  readonly onToggleContentLibrary: () => void;
}

export function EditorToolbar({
  showAI,
  onToggleAI,
  showATS,
  onToggleATS,
  showContentLibrary,
  onToggleContentLibrary,
}: EditorToolbarProps) {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const exportPDF = useExportPDF();
  const exportDOCX = useExportDOCX();
  const { undo, redo, canUndo, canRedo } = useUndoRedo();
  const [showExportMenu, setShowExportMenu] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const exportBtnRef = useRef<HTMLButtonElement>(null);

  // Close export menu when clicking outside
  useEffect(() => {
    if (!showExportMenu) return;
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        dropdownRef.current?.contains(target) ||
        exportBtnRef.current?.contains(target)
      )
        return;
      setShowExportMenu(false);
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [showExportMenu]);

  const handleExportPDF = useCallback(() => {
    if (!resume) return;

    exportPDF.mutate(resume.id, {
      onSuccess: (blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${resume.title || "resume"}.pdf`;
        a.click();
        setTimeout(() => URL.revokeObjectURL(url), 200);
        showSuccessNotification(t("resumeBuilder.toolbar.exportSuccess"));
      },
      onError: () => {
        showErrorNotification(t("resumeBuilder.toolbar.exportError"));
      },
    });
  }, [resume, exportPDF, t]);

  const handleExportDOCX = useCallback(() => {
    if (!resume) return;

    exportDOCX.mutate(resume.id, {
      onSuccess: (blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${resume.title || "resume"}.docx`;
        a.click();
        setTimeout(() => URL.revokeObjectURL(url), 200);
        showSuccessNotification(t("resumeBuilder.toolbar.exportDocxSuccess"));
      },
      onError: () => {
        showErrorNotification(t("resumeBuilder.toolbar.exportDocxError"));
      },
    });
  }, [resume, exportDOCX, t]);

  return (
    <div className="flex items-center gap-1 sm:gap-2">
      {/* Undo / Redo */}
      <Tooltip content={t("resumeBuilder.toolbar.undo")} side="bottom">
        <Button
          variant="outline"
          size="sm"
          className="shrink-0"
          onClick={undo}
          disabled={!canUndo}
          aria-label={t("resumeBuilder.toolbar.undo")}
        >
          <Undo2 className="h-4 w-4" />
        </Button>
      </Tooltip>
      <Tooltip content={t("resumeBuilder.toolbar.redo")} side="bottom">
        <Button
          variant="outline"
          size="sm"
          className="shrink-0"
          onClick={redo}
          disabled={!canRedo}
          aria-label={t("resumeBuilder.toolbar.redo")}
        >
          <Redo2 className="h-4 w-4" />
        </Button>
      </Tooltip>

      <div className="mx-0.5 h-6 w-px shrink-0 bg-border sm:mx-1" />

      {/* Export dropdown */}
      <div className="relative shrink-0">
        <Button
          ref={exportBtnRef}
          variant="outline"
          size="sm"
          onClick={() => setShowExportMenu(!showExportMenu)}
          disabled={exportPDF.isPending || exportDOCX.isPending || !resume}
        >
          {exportPDF.isPending || exportDOCX.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="hidden sm:inline">
                {t("resumeBuilder.toolbar.exporting")}
              </span>
            </>
          ) : (
            <>
              <Download className="h-4 w-4" />
              <span className="hidden sm:inline">
                {t("resumeBuilder.toolbar.export")}
              </span>
              <ChevronDown className="ml-1 h-3 w-3" />
            </>
          )}
        </Button>
        {showExportMenu && (
          <div
            ref={dropdownRef}
            className="absolute right-0 top-full z-50 mt-1 w-40 rounded-md border bg-popover p-1 shadow-md"
          >
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
              onClick={() => {
                setShowExportMenu(false);
                handleExportPDF();
              }}
            >
              <Download className="h-4 w-4" />
              {t("resumeBuilder.toolbar.exportPdf")}
            </button>
            <button
              type="button"
              className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
              onClick={() => {
                setShowExportMenu(false);
                handleExportDOCX();
              }}
            >
              <FileText className="h-4 w-4" />
              {t("resumeBuilder.toolbar.exportDocx")}
            </button>
          </div>
        )}
      </div>

      {/* AI Assistant toggle */}
      <Tooltip content={t("resumeBuilder.toolbar.aiAssistant")} side="bottom">
        <Button
          variant={showAI ? "default" : "outline"}
          size="sm"
          className="shrink-0"
          onClick={onToggleAI}
          aria-label={t("resumeBuilder.toolbar.aiAssistant")}
        >
          <Sparkles className="h-4 w-4" />
          <span className="hidden sm:inline">
            {t("resumeBuilder.toolbar.aiAssistant")}
          </span>
        </Button>
      </Tooltip>

      {/* ATS Check toggle */}
      <Tooltip content={t("resumeBuilder.toolbar.atsCheck")} side="bottom">
        <Button
          variant={showATS ? "default" : "outline"}
          size="sm"
          className="shrink-0"
          onClick={onToggleATS}
          aria-label={t("resumeBuilder.toolbar.atsCheck")}
        >
          <ShieldCheck className="h-4 w-4" />
          <span className="hidden sm:inline">
            {t("resumeBuilder.toolbar.atsCheck")}
          </span>
        </Button>
      </Tooltip>

      {/* Content Library toggle */}
      <Tooltip
        content={t("resumeBuilder.toolbar.contentLibrary")}
        side="bottom"
      >
        <Button
          variant={showContentLibrary ? "default" : "outline"}
          size="sm"
          className="shrink-0"
          onClick={onToggleContentLibrary}
          aria-label={t("resumeBuilder.toolbar.contentLibrary")}
        >
          <BookOpen className="h-4 w-4" />
          <span className="hidden sm:inline">
            {t("resumeBuilder.toolbar.contentLibrary")}
          </span>
        </Button>
      </Tooltip>
    </div>
  );
}
