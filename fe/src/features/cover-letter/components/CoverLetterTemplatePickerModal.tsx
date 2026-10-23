import { useTranslation } from "react-i18next";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/shared/ui/Dialog";
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
  if (!coverLetter) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }} className="max-w-4xl">
      <DialogContent onClose={onClose} className="flex max-h-[90dvh] max-w-none flex-col overflow-hidden p-0">
        <DialogHeader className="shrink-0 border-b px-6 py-5 pr-14 text-left">
          <DialogTitle>{t("coverLetter.chooseTemplate")}</DialogTitle>
        </DialogHeader>

        {/* Template grid */}
        <div className="overflow-y-auto p-6">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {COVER_LETTER_TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.id}
                onClick={() => {
                  updateField("template", tmpl.id);
                  onClose();
                }}
                className={cn(
                  "flex min-h-11 flex-col items-center rounded-xl border-2 p-3 transition-colors hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
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
      </DialogContent>
    </Dialog>
  );
}
