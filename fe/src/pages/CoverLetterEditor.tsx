import { useCallback, useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  ChevronDown,
  Download,
  Eye,
  Loader2,
  Redo2,
  Sparkles,
  Undo2,
  X,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Sheet } from "@/shared/ui/Sheet";
import { coverLetterService } from "@/services/coverLetterService";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { useAutoSaveCoverLetter } from "@/features/cover-letter/hooks/useAutoSaveCoverLetter";
import { useExportCoverLetterPDF } from "@/features/cover-letter/hooks/useExportCoverLetterPDF";
import { useExportCoverLetterDOCX } from "@/features/cover-letter/hooks/useExportCoverLetterDOCX";
import { useUndoRedoCoverLetter } from "@/features/cover-letter/hooks/useUndoRedoCoverLetter";
import {
  CoverLetterPreview,
  CoverLetterFullscreenPreview,
} from "@/features/cover-letter/components/CoverLetterPreview";
import { CoverLetterSaveIndicator } from "@/features/cover-letter/components/CoverLetterSaveIndicator";
import { CoverLetterAIPanel } from "@/features/cover-letter/components/CoverLetterAIPanel";
import { CoverLetterTemplateThumbnail } from "@/features/cover-letter/components/CoverLetterTemplateThumbnail";
import {
  CoverLetterTemplatePickerModal,
  COVER_LETTER_TEMPLATES,
} from "@/features/cover-letter/components/CoverLetterTemplatePickerModal";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { cn } from "@/shared/lib/utils";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";

const HEX_PARTIAL_REGEX = /^#[0-9a-fA-F]{0,6}$/;
const HEX_COMPLETE_REGEX = /^#[0-9a-fA-F]{6}$/;

const FREE_FONTS = ["Georgia", "Arial", "Times New Roman"];
const PREMIUM_FONTS = [
  "Roboto",
  "Open Sans",
  "Lato",
  "Montserrat",
  "Poppins",
  "Inter",
  "Merriweather",
  "PT Serif",
  "Source Sans Pro",
  "Nunito",
  "Raleway",
  "Playfair Display",
];

const FONT_SIZES = [
  { label: "S", value: 10 },
  { label: "M", value: 12 },
  { label: "L", value: 14 },
  { label: "XL", value: 16 },
] as const;

const PRESET_COLORS = [
  "#2563eb",
  "#1d4ed8",
  "#3b82f6",
  "#0ea5e9",
  "#0891b2",
  "#059669",
  "#16a34a",
  "#65a30d",
  "#ca8a04",
  "#ea580c",
  "#dc2626",
  "#e11d48",
  "#db2777",
  "#9333ea",
  "#7c3aed",
  "#4f46e5",
  "#1e293b",
  "#334155",
  "#475569",
  "#64748b",
  "#78716c",
  "#57534e",
  "#44403c",
  "#292524",
  "#171717",
  "#000000",
  "#0f172a",
  "#1e3a5f",
  "#14532d",
  "#7f1d1d",
];

export default function CoverLetterEditorPage() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const setCoverLetter = useCoverLetterStore((s) => s.setCoverLetter);
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);
  const updateFields = useCoverLetterStore((s) => s.updateFields);
  const [showAI, setShowAI] = useState(false);
  const [showFullPreview, setShowFullPreview] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showColorDropdown, setShowColorDropdown] = useState(false);
  const [localColorInput, setLocalColorInput] = useState("");
  const colorDropdownRef = useRef<HTMLDivElement>(null);

  const isDesktop = useMediaQuery("(min-width: 1024px)");

  usePageMeta({ titleKey: "coverLetter.editor" });
  useAutoSaveCoverLetter();
  const exportPDF = useExportCoverLetterPDF();
  const exportDOCX = useExportCoverLetterDOCX();
  const { undo, redo, canUndo, canRedo } = useUndoRedoCoverLetter();

  const { isLoading, error, data } = useQuery({
    queryKey: ["cover-letter", id],
    queryFn: () => coverLetterService.getById(id!),
    enabled: !!id,
  });

  useEffect(() => {
    if (data) {
      setCoverLetter(data);
      useCoverLetterStore.temporal.getState().clear();
    }
  }, [data, setCoverLetter]);

  useEffect(() => {
    if (!showColorDropdown) return;
    const handler = (e: MouseEvent) => {
      if (
        colorDropdownRef.current &&
        !colorDropdownRef.current.contains(e.target as Node)
      ) {
        setShowColorDropdown(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showColorDropdown]);

  const handleToggleAI = useCallback(() => setShowAI((v) => !v), []);
  const handleCloseAI = useCallback(() => setShowAI(false), []);
  const handleTogglePreview = useCallback(
    () => setShowFullPreview((v) => !v),
    [],
  );
  const handleClosePreview = useCallback(() => setShowFullPreview(false), []);

  const handleExportPDF = useCallback(() => {
    if (!coverLetter) return;
    exportPDF.mutate(coverLetter.id, {
      onSuccess: (blob) => {
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `${coverLetter.title || "cover-letter"}.pdf`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
        showSuccessNotification(t("coverLetter.exported"));
      },
      onError: (err) => {
        showErrorNotification(
          err instanceof Error ? err.message : t("common.error"),
        );
      },
    });
  }, [coverLetter, exportPDF, t]);

  const handleExportDOCX = useCallback(() => {
    if (!coverLetter) return;
    exportDOCX.mutate(coverLetter.id, {
      onSuccess: (blob) => {
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `${coverLetter.title || "cover-letter"}.docx`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
        showSuccessNotification(t("coverLetter.exportedDOCX"));
      },
      onError: (err) => {
        showErrorNotification(
          err instanceof Error ? err.message : t("common.error"),
        );
      },
    });
  }, [coverLetter, exportDOCX, t]);

  if (isLoading) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !coverLetter) {
    return (
      <div className="flex h-[calc(100vh-8rem)] flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">{t("coverLetter.notFound")}</p>
        <Button
          variant="outline"
          onClick={() => navigate("/app/cover-letters")}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          {t("coverLetter.backToList")}
        </Button>
      </div>
    );
  }

  const currentTemplate =
    COVER_LETTER_TEMPLATES.find((tmpl) => tmpl.id === coverLetter.template) ??
    COVER_LETTER_TEMPLATES[0];

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col">
      {/* Top bar: back + title + actions */}
      <div className="flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/app/cover-letters")}
            aria-label={t("coverLetter.backToList")}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <Input
            value={coverLetter.title}
            onChange={(e) => updateField("title", e.target.value)}
            className="h-8 w-48 border-transparent bg-transparent text-lg font-semibold hover:border-input focus:border-input sm:w-64"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleTogglePreview}
            aria-label={t("coverLetter.preview")}
          >
            <Eye className="h-4 w-4" />
            <span className="hidden sm:inline">{t("coverLetter.preview")}</span>
          </Button>
          <Button
            variant={showAI ? "default" : "outline"}
            size="sm"
            onClick={handleToggleAI}
            aria-label={t("coverLetter.ai.title")}
          >
            <Sparkles className="h-4 w-4" />
            <span className="hidden sm:inline">
              {t("coverLetter.ai.title")}
            </span>
          </Button>
          <CoverLetterSaveIndicator />
        </div>
      </div>

      {/* Design toolbar */}
      <div className="relative flex flex-wrap items-center gap-2 border-b px-4 py-2">
        {/* Template picker — opens modal */}
        <button
          onClick={() => setShowTemplateModal(true)}
          className="flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors hover:bg-accent"
        >
          <CoverLetterTemplateThumbnail
            templateId={coverLetter.template}
            accentColor={coverLetter.primary_color}
          />
          <span className="hidden sm:inline">
            {t(currentTemplate.labelKey)}
          </span>
          <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
        </button>

        <div className="h-6 w-px bg-border" />

        {/* Font family */}
        <select
          value={coverLetter.font_family}
          onChange={(e) => updateField("font_family", e.target.value)}
          className="h-8 rounded-md border border-input bg-background px-2 text-sm"
        >
          <optgroup label={t("coverLetter.design.freeFonts")}>
            {FREE_FONTS.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </optgroup>
          <optgroup label={t("coverLetter.design.premiumFonts")}>
            {PREMIUM_FONTS.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </optgroup>
        </select>

        <div className="h-6 w-px bg-border" />

        {/* Font size */}
        <div className="flex">
          {FONT_SIZES.map((size) => (
            <button
              key={size.value}
              onClick={() => updateFields({ font_size: size.value })}
              className={cn(
                "border px-2.5 py-1 text-xs font-medium transition-colors first:rounded-l-md last:rounded-r-md",
                (coverLetter.font_size ?? 12) === size.value
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-input hover:bg-accent",
              )}
            >
              {size.label}
            </button>
          ))}
        </div>

        <div className="h-6 w-px bg-border" />

        {/* Color picker */}
        <div ref={colorDropdownRef} className="relative">
          <button
            onClick={() => setShowColorDropdown((v) => !v)}
            className={cn(
              "flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm transition-colors hover:bg-accent",
              showColorDropdown && "border-primary bg-accent",
            )}
          >
            <span
              className="h-4 w-4 rounded-full border border-black/10"
              style={{ backgroundColor: coverLetter.primary_color }}
            />
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          </button>

          {showColorDropdown && (
            <div className="absolute left-0 top-full z-50 mt-1 w-64 rounded-lg border bg-popover p-3 shadow-lg">
              <div className="flex flex-wrap gap-1.5">
                {PRESET_COLORS.map((color) => (
                  <button
                    key={color}
                    onClick={() => {
                      updateFields({ primary_color: color });
                      setShowColorDropdown(false);
                    }}
                    className={cn(
                      "h-6 w-6 rounded-full border-2 transition-transform hover:scale-110",
                      coverLetter.primary_color === color
                        ? "border-foreground ring-2 ring-primary ring-offset-1"
                        : "border-transparent",
                    )}
                    style={{ backgroundColor: color }}
                    aria-label={color}
                  />
                ))}
              </div>
              <div className="mt-2 flex items-center gap-2">
                <input
                  type="color"
                  value={coverLetter.primary_color}
                  onChange={(e) =>
                    updateFields({ primary_color: e.target.value })
                  }
                  className="h-7 w-8 cursor-pointer rounded border-none p-0"
                />
                <Input
                  value={localColorInput || coverLetter.primary_color}
                  onChange={(e) => {
                    const val = e.target.value;
                    if (!HEX_PARTIAL_REGEX.test(val)) return;
                    setLocalColorInput(val);
                    if (HEX_COMPLETE_REGEX.test(val)) {
                      updateFields({ primary_color: val });
                    }
                  }}
                  onFocus={() => setLocalColorInput(coverLetter.primary_color)}
                  onBlur={() => setLocalColorInput("")}
                  className="h-7 w-24 font-mono text-xs"
                  maxLength={7}
                />
              </div>
            </div>
          )}
        </div>

        <div className="h-6 w-px bg-border" />

        {/* Undo / Redo */}
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={undo}
          disabled={!canUndo}
          aria-label={t("coverLetter.undo")}
          title={t("coverLetter.undo")}
        >
          <Undo2 className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={redo}
          disabled={!canRedo}
          aria-label={t("coverLetter.redo")}
          title={t("coverLetter.redo")}
        >
          <Redo2 className="h-4 w-4" />
        </Button>

        <div className="h-6 w-px bg-border" />

        {/* Export PDF */}
        <Button
          variant="outline"
          size="sm"
          onClick={handleExportPDF}
          disabled={exportPDF.isPending}
        >
          <Download className="h-4 w-4" />
          <span className="hidden sm:inline">{t("coverLetter.exportPDF")}</span>
        </Button>

        {/* Export DOCX */}
        <Button
          variant="outline"
          size="sm"
          onClick={handleExportDOCX}
          disabled={exportDOCX.isPending}
        >
          <Download className="h-4 w-4" />
          <span className="hidden sm:inline">
            {t("coverLetter.exportDOCX")}
          </span>
        </Button>
      </div>

      {/* Main area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Preview */}
        <div
          className={cn(
            "flex-1 overflow-y-auto bg-muted/30",
            showAI && isDesktop && "flex-initial w-[calc(100%-320px)]",
          )}
        >
          <CoverLetterPreview editable />
        </div>

        {/* AI Panel — desktop (>= lg) */}
        {showAI && isDesktop && (
          <div className="w-[320px] min-w-[320px] overflow-y-auto border-l bg-background">
            <div className="flex items-center justify-between border-b px-4 py-2">
              <h2 className="text-sm font-semibold">
                {t("coverLetter.ai.title")}
              </h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleCloseAI}
                aria-label={t("common.close")}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="p-4">
              <CoverLetterAIPanel />
            </div>
          </div>
        )}
      </div>

      {/* Fullscreen Preview */}
      <CoverLetterFullscreenPreview
        open={showFullPreview}
        onClose={handleClosePreview}
      />

      {/* Template picker modal */}
      <CoverLetterTemplatePickerModal
        isOpen={showTemplateModal}
        onClose={() => setShowTemplateModal(false)}
      />

      {/* Mobile sheet for AI panel (< lg) */}
      {!isDesktop && (
        <Sheet
          open={showAI}
          onOpenChange={(open) => {
            if (!open) handleCloseAI();
          }}
          title={t("coverLetter.ai.title")}
        >
          <CoverLetterAIPanel />
        </Sheet>
      )}
    </div>
  );
}
