import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { cn } from "@/shared/lib/utils";
import { TEMPLATE_LIST } from "../../lib/templateRegistry";
import { TemplateThumbnail } from "./TemplateThumbnail";
import { X } from "lucide-react";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";

/** Larger thumbnail width for the fullscreen modal */
const MODAL_THUMB_WIDTH = 160;
// Four 160px thumbnails in a phone-wide dialog overlapped their neighbours and
// the dialog edge; two columns of 120px fit with room for the card padding.
const PHONE_THUMB_WIDTH = 120;
const PHONE_VIEWPORT = "(max-width: 639px)";

interface TemplatePickerPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
}

export function TemplatePickerPopover({
  isOpen,
  onClose,
}: TemplatePickerPopoverProps) {
  const { t } = useTranslation();
  const isPhone = useMediaQuery(PHONE_VIEWPORT);
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);
  const backdropRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

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
        aria-labelledby="template-picker-heading"
        className="relative mx-4 flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-xl bg-background shadow-2xl"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b px-6 py-4">
          <h2 id="template-picker-heading" className="text-lg font-semibold">
            {t("resumeBuilder.design.template")}
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
        <div className="overflow-y-auto p-4 sm:p-6">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-5 lg:grid-cols-4">
            {TEMPLATE_LIST.map((tmpl) => (
              <button
                key={tmpl.id}
                onClick={() => {
                  updateDesign({ template_id: tmpl.id });
                  onClose();
                }}
                className={cn(
                  "flex min-w-0 flex-col items-center rounded-xl border-2 p-3 transition-all hover:shadow-md sm:p-4",
                  resume?.template_id === tmpl.id
                    ? "border-primary bg-primary/5 shadow-md"
                    : "border-border hover:border-primary/50",
                )}
              >
                <TemplateThumbnail
                  templateId={tmpl.id}
                  width={isPhone ? PHONE_THUMB_WIDTH : MODAL_THUMB_WIDTH}
                />
                <span className="mt-3 text-center text-sm font-medium">
                  {t(tmpl.nameKey)}
                </span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
