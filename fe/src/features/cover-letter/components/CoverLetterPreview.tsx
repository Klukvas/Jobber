import { useRef, useState, useEffect } from "react";
import { useTranslation, getI18n } from "react-i18next";
import { Plus, X } from "lucide-react";
import { EditableField } from "@/features/resume-builder/components/inline/EditableField";
import { EditableTextarea } from "@/features/resume-builder/components/inline/EditableTextarea";
import { useCoverLetterStore } from "@/stores/coverLetterStore";

const A4_WIDTH_PX = 793; // 210mm at 96dpi
const A4_HEIGHT_PX = 1122; // 297mm at 96dpi

interface CoverLetterTemplateProps {
  readonly editable?: boolean;
}

/* ------------------------------------------------------------------ */
/*  Shared helpers                                                     */
/* ------------------------------------------------------------------ */

/** Single-line field that hides when empty & not editable */
function CLField({
  value,
  onChange,
  editable,
  placeholder = "",
  className,
  style,
}: {
  readonly value: string;
  readonly onChange: (v: string) => void;
  readonly editable: boolean;
  readonly placeholder?: string;
  readonly className?: string;
  readonly style?: React.CSSProperties;
}) {
  if (!editable && !value) return null;
  return (
    <EditableField
      value={value}
      onChange={onChange}
      editable={editable}
      placeholder={placeholder}
      className={className}
      style={style}
      as="p"
    />
  );
}

/** Multi-line field that hides when empty & not editable */
function CLTextArea({
  value,
  onChange,
  editable,
  placeholder = "",
  className,
  style,
}: {
  readonly value: string;
  readonly onChange: (v: string) => void;
  readonly editable: boolean;
  readonly placeholder?: string;
  readonly className?: string;
  readonly style?: React.CSSProperties;
}) {
  if (!editable && !value) return null;
  return (
    <EditableTextarea
      value={value}
      onChange={onChange}
      editable={editable}
      placeholder={placeholder}
      className={className}
      style={style}
    />
  );
}

/** Paragraphs block shared by all templates */
function CoverLetterParagraphs({ editable }: { readonly editable: boolean }) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateParagraph = useCoverLetterStore((s) => s.updateParagraph);
  const addParagraph = useCoverLetterStore((s) => s.addParagraph);
  const removeParagraph = useCoverLetterStore((s) => s.removeParagraph);

  if (!coverLetter) return null;

  return (
    <>
      {coverLetter.paragraphs.map((paragraph, index) => {
        const key = `${index}-${paragraph.slice(0, 20)}`;
        return editable ? (
          <div key={key} className="group/para relative mb-3">
            <EditableTextarea
              value={paragraph}
              onChange={(v) => updateParagraph(index, v)}
              editable
              placeholder={t("coverLetter.placeholders.paragraph", {
                number: index + 1,
              })}
              className="break-words pr-5 leading-relaxed text-gray-700"
            />
            {coverLetter.paragraphs.length > 1 && (
              <button
                onMouseDown={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  removeParagraph(index);
                }}
                className="absolute right-0 top-0 hidden rounded p-0.5 text-red-400 transition-colors hover:bg-red-50 hover:text-red-600 group-hover/para:block"
                aria-label={t("coverLetter.removeParagraph")}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        ) : (
          <p
            key={key}
            className="mb-3 break-words leading-relaxed text-gray-700"
          >
            {paragraph || "\u00A0"}
          </p>
        );
      })}
      {editable && (
        <button
          onMouseDown={(e) => {
            e.preventDefault();
            addParagraph();
          }}
          className="mt-1 flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
        >
          <Plus className="h-3 w-3" />
          {t("coverLetter.addParagraph")}
        </button>
      )}
    </>
  );
}

/** Formatted date line */
function CoverLetterDate({ className }: { readonly className?: string }) {
  return (
    <p className={className}>
      {new Date().toLocaleDateString(getI18n().language, {
        year: "numeric",
        month: "long",
        day: "numeric",
      })}
    </p>
  );
}

/* ------------------------------------------------------------------ */
/*  Template components                                                */
/* ------------------------------------------------------------------ */

function ProfessionalCoverLetter({
  editable = false,
}: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div className="mb-6 border-b-2 pb-4" style={{ borderColor: color }}>
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-xs text-gray-700"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-600"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs font-semibold text-gray-800"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-600"
        />
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 font-semibold text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="font-semibold text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

function ModernCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div className="mb-6 flex gap-4">
        <div
          className="w-1 shrink-0 rounded"
          style={{ backgroundColor: color }}
        />
        <div>
          <CLField
            value={coverLetter.company_name}
            onChange={(v) => updateField("company_name", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.companyName")}
            className="text-sm font-bold"
            style={{ color }}
          />
          <CLField
            value={coverLetter.recipient_name}
            onChange={(v) => updateField("recipient_name", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.recipientName")}
            className="text-xs text-gray-700"
          />
          <CLField
            value={coverLetter.recipient_title}
            onChange={(v) => updateField("recipient_title", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.recipientTitle")}
            className="text-xs text-gray-500"
          />
          <CLTextArea
            value={coverLetter.company_address}
            onChange={(v) => updateField("company_address", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.companyAddress")}
            className="text-xs text-gray-500"
          />
        </div>
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-400" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4"
        style={{ color }}
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            style={{ color }}
          />
        </div>
      )}
    </div>
  );
}

function MinimalCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  return (
    <div>
      <div className="mb-6">
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-xs text-gray-800"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-600"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs text-gray-600"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-500"
        />
      </div>

      <CoverLetterDate className="mb-6 text-xs text-gray-400" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-8">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

function ExecutiveCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div
        className="mb-6 rounded-lg px-6 py-5"
        style={{ backgroundColor: color }}
      >
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-sm font-bold text-white"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-white/80"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs font-semibold text-white/90"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-white/70"
        />
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 font-bold text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="font-bold text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

function CreativeCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div className="flex gap-6">
      <div
        className="w-36 shrink-0 rounded-lg px-4 py-5"
        style={{ backgroundColor: color }}
      >
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="mb-1 text-xs font-bold text-white"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="mb-3 text-xs text-white/80"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="mb-1 text-xs font-semibold text-white/90"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-white/70"
        />
      </div>

      <div className="flex-1">
        <CoverLetterDate className="mb-4 text-xs text-gray-400" />

        <CLField
          value={coverLetter.greeting}
          onChange={(v) => updateField("greeting", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.greeting")}
          className="mb-4 font-semibold"
          style={{ color }}
        />

        <CoverLetterParagraphs editable={editable} />

        {(editable || coverLetter.closing) && (
          <div className="mt-6">
            <CLField
              value={coverLetter.closing}
              onChange={(v) => updateField("closing", v)}
              editable={editable}
              placeholder={t("coverLetter.placeholders.closing")}
              className="font-semibold"
              style={{ color }}
            />
          </div>
        )}
      </div>
    </div>
  );
}

function ClassicCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  return (
    <div>
      <hr className="mb-6 border-t border-gray-300" />
      <div className="mb-6 text-center">
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-sm font-semibold text-gray-800"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-600"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs text-gray-600"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-500"
        />
      </div>

      <CoverLetterDate className="mb-4 text-center text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="text-gray-800"
          />
        </div>
      )}
      <hr className="mt-8 border-t border-gray-300" />
    </div>
  );
}

function ElegantCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div
        className="mx-[-40px] mt-[-40px] mb-6 h-[3px]"
        style={{ backgroundColor: color }}
      />

      <div className="mb-6">
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-xs text-gray-700"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-600"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs font-semibold text-gray-800"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-500"
        />
      </div>

      <CoverLetterDate className="mb-4 text-right text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 italic text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            style={{ color }}
          />
        </div>
      )}
    </div>
  );
}

function BoldCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div
        className="-mx-[40px] -mt-[40px] mb-6 flex h-16 items-end px-10 pb-3"
        style={{ backgroundColor: color }}
      >
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-lg font-bold text-white"
        />
      </div>

      <div className="mb-6">
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-600"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs font-semibold text-gray-800"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-500"
        />
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 font-bold text-gray-800"
      />

      <div className="border-l-4 pl-4" style={{ borderColor: color }}>
        <CoverLetterParagraphs editable={editable} />
      </div>

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="font-bold text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

function SimpleCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  return (
    <div>
      <div className="mb-6">
        <CLField
          value={coverLetter.recipient_name}
          onChange={(v) => updateField("recipient_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientName")}
          className="text-xs text-gray-800"
        />
        <CLField
          value={coverLetter.recipient_title}
          onChange={(v) => updateField("recipient_title", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.recipientTitle")}
          className="text-xs text-gray-800"
        />
        <CLField
          value={coverLetter.company_name}
          onChange={(v) => updateField("company_name", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyName")}
          className="text-xs text-gray-800"
        />
        <CLTextArea
          value={coverLetter.company_address}
          onChange={(v) => updateField("company_address", v)}
          editable={editable}
          placeholder={t("coverLetter.placeholders.companyAddress")}
          className="text-xs text-gray-800"
        />
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-800" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

function CorporateCoverLetter({ editable = false }: CoverLetterTemplateProps) {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);

  if (!coverLetter) return null;

  const color = coverLetter.primary_color;

  return (
    <div>
      <div
        className="mb-6 flex justify-between border-b-2 pb-4"
        style={{ borderColor: color }}
      >
        <div>
          <CLField
            value={coverLetter.recipient_name}
            onChange={(v) => updateField("recipient_name", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.recipientName")}
            className="text-xs font-semibold text-gray-800"
          />
          <CLField
            value={coverLetter.recipient_title}
            onChange={(v) => updateField("recipient_title", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.recipientTitle")}
            className="text-xs text-gray-600"
          />
        </div>
        <div className="text-right">
          <CLField
            value={coverLetter.company_name}
            onChange={(v) => updateField("company_name", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.companyName")}
            className="text-xs font-semibold text-gray-800"
          />
          <CLTextArea
            value={coverLetter.company_address}
            onChange={(v) => updateField("company_address", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.companyAddress")}
            className="text-xs text-gray-500"
          />
        </div>
      </div>

      <CoverLetterDate className="mb-4 text-xs text-gray-500" />

      <CLField
        value={coverLetter.greeting}
        onChange={(v) => updateField("greeting", v)}
        editable={editable}
        placeholder={t("coverLetter.placeholders.greeting")}
        className="mb-4 font-semibold text-gray-800"
      />

      <CoverLetterParagraphs editable={editable} />

      {(editable || coverLetter.closing) && (
        <div className="mt-6">
          <CLField
            value={coverLetter.closing}
            onChange={(v) => updateField("closing", v)}
            editable={editable}
            placeholder={t("coverLetter.placeholders.closing")}
            className="font-semibold text-gray-800"
          />
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Template registry                                                  */
/* ------------------------------------------------------------------ */

const TEMPLATE_MAP: Record<
  string,
  React.ComponentType<CoverLetterTemplateProps>
> = {
  professional: ProfessionalCoverLetter,
  modern: ModernCoverLetter,
  minimal: MinimalCoverLetter,
  executive: ExecutiveCoverLetter,
  creative: CreativeCoverLetter,
  classic: ClassicCoverLetter,
  elegant: ElegantCoverLetter,
  bold: BoldCoverLetter,
  simple: SimpleCoverLetter,
  corporate: CorporateCoverLetter,
};

/* ------------------------------------------------------------------ */
/*  Preview components                                                 */
/* ------------------------------------------------------------------ */

interface CoverLetterPreviewProps {
  readonly editable?: boolean;
}

function PageBreakIndicator() {
  const { t } = useTranslation();
  return (
    <div
      className="pointer-events-none absolute left-0 right-0"
      style={{ top: A4_HEIGHT_PX }}
    >
      <div className="relative flex items-center justify-center">
        <div className="absolute inset-x-0 h-px bg-red-400/60" />
        <span className="relative rounded bg-red-400 px-3 py-1 text-xs text-white">
          {t("coverLetter.pageBreakWarning")}
        </span>
      </div>
    </div>
  );
}

export function CoverLetterPreview({
  editable = false,
}: CoverLetterPreviewProps) {
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const containerRef = useRef<HTMLDivElement>(null);
  const pageRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(editable ? 1 : 0.65);
  const [overflows, setOverflows] = useState(false);

  useEffect(() => {
    if (!editable || !containerRef.current) return;
    const updateScale = () => {
      if (!containerRef.current) return;
      const available = containerRef.current.clientWidth - 48;
      const newScale = Math.min(available / A4_WIDTH_PX, 1);
      setScale(Math.max(newScale, 0.4));
    };
    updateScale();
    const observer = new ResizeObserver(updateScale);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [editable]);

  useEffect(() => {
    if (!pageRef.current) return;
    const el = pageRef.current;
    const observer = new ResizeObserver(() => {
      setOverflows(el.scrollHeight > A4_HEIGHT_PX + 2);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  if (!coverLetter) return null;

  const TemplateComponent =
    TEMPLATE_MAP[coverLetter.template] ?? ProfessionalCoverLetter;

  const pageStyle: React.CSSProperties = {
    width: "210mm",
    minHeight: A4_HEIGHT_PX,
    fontFamily: coverLetter.font_family,
    fontSize: `${coverLetter.font_size ?? 12}px`,
    padding: 40,
    overflowWrap: "break-word",
    wordBreak: "break-word" as React.CSSProperties["wordBreak"],
  };

  return (
    <div ref={containerRef} className="flex justify-center p-6">
      <div
        className="origin-top"
        style={{ transform: `scale(${scale})`, transformOrigin: "top center" }}
      >
        <div className="relative">
          <div
            ref={pageRef}
            className="bg-white text-black shadow-lg"
            style={pageStyle}
          >
            <TemplateComponent editable={editable} />
          </div>
          {overflows && <PageBreakIndicator />}
        </div>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Fullscreen preview                                                 */
/* ------------------------------------------------------------------ */

function CloseButton({ onClose }: { readonly onClose: () => void }) {
  const { t } = useTranslation();
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onClose();
      }}
      aria-label={t("common.close")}
      className="fixed right-4 top-4 z-50 flex h-10 w-10 items-center justify-center rounded-full bg-black/50 text-white transition-colors hover:bg-black/70"
    >
      <X className="h-5 w-5" />
    </button>
  );
}

interface CoverLetterFullscreenPreviewProps {
  readonly open: boolean;
  readonly onClose: () => void;
}

export function CoverLetterFullscreenPreview({
  open,
  onClose,
}: CoverLetterFullscreenPreviewProps) {
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const containerRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(0.65);

  useEffect(() => {
    if (!open || !containerRef.current) return;

    const updateScale = () => {
      if (!containerRef.current) return;
      const available = containerRef.current.clientWidth - 48;
      const newScale = Math.min(available / A4_WIDTH_PX, 1);
      setScale(Math.max(newScale, 0.4));
    };

    updateScale();
    const observer = new ResizeObserver(updateScale);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKey);

    return () => {
      document.body.style.overflow = prev;
      document.removeEventListener("keydown", handleKey);
    };
  }, [open, onClose]);

  if (!open || !coverLetter) return null;

  const TemplateComponent =
    TEMPLATE_MAP[coverLetter.template] ?? ProfessionalCoverLetter;

  const pageStyle: React.CSSProperties = {
    width: "210mm",
    minHeight: A4_HEIGHT_PX,
    fontFamily: coverLetter.font_family,
    fontSize: `${coverLetter.font_size ?? 12}px`,
    padding: 40,
    overflowWrap: "break-word",
    wordBreak: "break-word" as React.CSSProperties["wordBreak"],
  };

  return (
    <div className="fixed inset-0 z-50 flex flex-col" onClick={onClose}>
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" />
      <CloseButton onClose={onClose} />
      <div
        ref={containerRef}
        className="relative z-10 flex-1 overflow-y-auto py-8"
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className="mx-auto origin-top"
          style={{
            transform: `scale(${scale})`,
            transformOrigin: "top center",
            width: "210mm",
          }}
        >
          <div className="bg-white text-black shadow-2xl" style={pageStyle}>
            <TemplateComponent />
          </div>
        </div>
      </div>
    </div>
  );
}
