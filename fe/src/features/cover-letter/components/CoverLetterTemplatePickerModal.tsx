import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { X } from "lucide-react";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { cn } from "@/shared/lib/utils";
import { CoverLetterTemplateThumbnail } from "./CoverLetterTemplateThumbnail";

export const COVER_LETTER_TEMPLATES = [
  { id: "professional", labelKey: "coverLetter.templates.professional" },
  { id: "modern", labelKey: "coverLetter.templates.modern" },
  { id: "minimal", labelKey: "coverLetter.templates.minimal" },
  { id: "executive", labelKey: "coverLetter.templates.executive" },
  { id: "creative", labelKey: "coverLetter.templates.creative" },
  { id: "classic", labelKey: "coverLetter.templates.classic" },
  { id: "elegant", labelKey: "coverLetter.templates.elegant" },
  { id: "bold", labelKey: "coverLetter.templates.bold" },
  { id: "simple", labelKey: "coverLetter.templates.simple" },
  { id: "corporate", labelKey: "coverLetter.templates.corporate" },
] as const;

interface CoverLetterTemplatePickerModalProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
}

export function CoverLetterTemplatePickerModal({
  isOpen,
  onClose,
}: CoverLetterTemplatePickerModalProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);
  const backdropRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [isOpen, onClose]);

  if (!isOpen || !coverLetter) return null;

  return (
    <div
      ref={backdropRef}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={(e) => {
        if (e.target === backdropRef.current) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cl-template-picker-heading"
        className="relative mx-4 flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-xl bg-background shadow-2xl"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b px-6 py-4">
          <h2
            id="cl-template-picker-heading"
            className="text-lg font-semibold"
          >
            {t("coverLetter.chooseTemplate")}
          </h2>
          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-lg hover:bg-muted"
            aria-label={t("common.close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Template grid */}
        <div className="overflow-y-auto p-6">
          <div className="grid grid-cols-5 gap-5">
            {COVER_LETTER_TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.id}
                onClick={() => {
                  updateField("template", tmpl.id);
                  onClose();
                }}
                className={cn(
                  "flex flex-col items-center rounded-xl border-2 p-4 transition-all hover:shadow-md",
                  coverLetter.template === tmpl.id
                    ? "border-primary bg-primary/5 shadow-md"
                    : "border-border hover:border-primary/50",
                )}
              >
                <div className="rounded border shadow-sm">
                  <CoverLetterTemplateThumbnail
                    templateId={tmpl.id}
                    accentColor={coverLetter.primary_color}
                    size="lg"
                  />
                </div>
                <span className="mt-3 text-sm font-medium">
                  {t(tmpl.labelKey)}
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
