import { useId } from "react";
import { useTranslation } from "react-i18next";
import { Plus, X } from "lucide-react";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Button } from "@/shared/ui/Button";

const FIELDS = ["recipient_name", "recipient_title", "company_name", "company_address", "greeting", "closing"] as const;
const LABELS = {
  recipient_name: "recipientName", recipient_title: "recipientTitle",
  company_name: "companyName", company_address: "companyAddress",
  greeting: "greeting", closing: "closing",
} as const;

/** A fluid editing surface; document typography is reserved for the A4 preview. */
export function CoverLetterTextEditor() {
  const { t } = useTranslation();
  const prefix = useId();
  const { coverLetter, paragraphIds, updateField, updateParagraph, addParagraph, removeParagraph } = useCoverLetterStore();
  if (!coverLetter) return null;
  const renderField = (field: typeof FIELDS[number]) => (
    <div key={field} className="space-y-1.5">
      <label htmlFor={`${prefix}-${field}`} className="text-sm font-medium">
        {t(`coverLetter.fields.${LABELS[field]}`)}
      </label>
      {field === "company_address" ? (
        <Textarea id={`${prefix}-${field}`} value={coverLetter[field]} onChange={(event) => updateField(field, event.target.value)} className="min-h-11 text-base" />
      ) : (
        <Input id={`${prefix}-${field}`} value={coverLetter[field]} onChange={(event) => updateField(field, event.target.value)} className="min-h-11 text-base" />
      )}
    </div>
  );
  return (
    <div className="space-y-5 p-4 pb-[max(2rem,env(safe-area-inset-bottom))]" data-cover-letter-text-editor>
      {FIELDS.filter((field) => field !== "closing").map(renderField)}
      {coverLetter.paragraphs.map((paragraph, index) => (
        <div key={paragraphIds[index]} className="space-y-1.5">
          <div className="flex items-center justify-between gap-2">
            <label htmlFor={`${prefix}-${paragraphIds[index]}`} className="text-sm font-medium">
              {t("coverLetter.fields.paragraphs")} {index + 1}
            </label>
            {coverLetter.paragraphs.length > 1 && (
              <Button variant="ghost" size="icon" onClick={() => removeParagraph(index)} aria-label={t("coverLetter.removeParagraph")}><X className="h-4 w-4" /></Button>
            )}
          </div>
          <Textarea id={`${prefix}-${paragraphIds[index]}`} value={paragraph} onChange={(event) => updateParagraph(index, event.target.value)} rows={5} className="text-base" />
        </div>
      ))}
      <Button variant="outline" onClick={addParagraph}><Plus className="h-4 w-4" />{t("coverLetter.addParagraph")}</Button>
      {renderField("closing")}
    </div>
  );
}
