import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { cn } from "@/shared/lib/utils";
import { SectionOrderPanel } from "./SectionOrderPanel";
import { TEMPLATE_LIST } from "../lib/templateRegistry";

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

export function DesignPanel() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);

  if (!resume) return null;

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">
        {t("resumeBuilder.sections.design")}
      </h2>

      {/* Title */}
      <div className="space-y-1.5">
        <Label htmlFor="resume-title">{t("resumeBuilder.design.title")}</Label>
        <Input
          id="resume-title"
          value={resume.title}
          onChange={(e) => updateDesign({ title: e.target.value })}
        />
      </div>

      {/* Template selection */}
      <div className="space-y-2">
        <Label>{t("resumeBuilder.design.template")}</Label>
        <div className="grid grid-cols-3 gap-3">
          {TEMPLATE_LIST.map((tmpl) => (
            <button
              key={tmpl.id}
              onClick={() => updateDesign({ template_id: tmpl.id })}
              className={cn(
                "flex flex-col items-center rounded-lg border-2 p-3 transition-colors",
                resume.template_id === tmpl.id
                  ? "border-primary bg-primary/5"
                  : "border-border hover:border-primary/50",
              )}
            >
              <div className="mb-2 h-16 w-12 rounded border bg-white shadow-sm" />
              <span className="text-xs font-medium">{t(tmpl.nameKey)}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Color */}
      <div className="space-y-2">
        <Label>{t("resumeBuilder.design.color")}</Label>
        <div className="flex flex-wrap gap-2">
          {PRESET_COLORS.map((color) => (
            <button
              key={color}
              onClick={() => updateDesign({ primary_color: color })}
              className={cn(
                "h-7 w-7 rounded-full border-2 transition-transform hover:scale-110",
                resume.primary_color === color
                  ? "border-foreground ring-2 ring-primary ring-offset-2"
                  : "border-transparent",
              )}
              style={{ backgroundColor: color }}
              aria-label={color}
            />
          ))}
        </div>
        <div className="flex items-center gap-2">
          <Input
            type="color"
            value={resume.primary_color}
            onChange={(e) => updateDesign({ primary_color: e.target.value })}
            className="h-8 w-12 cursor-pointer p-0"
          />
          <Input
            value={resume.primary_color}
            onChange={(e) => updateDesign({ primary_color: e.target.value })}
            className="w-28 font-mono text-sm"
            maxLength={7}
          />
        </div>
      </div>

      {/* Font */}
      <div className="space-y-2">
        <Label htmlFor="font-family">{t("resumeBuilder.design.font")}</Label>
        <select
          id="font-family"
          value={resume.font_family}
          onChange={(e) => updateDesign({ font_family: e.target.value })}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          <optgroup label={t("resumeBuilder.design.freeFonts")}>
            {FREE_FONTS.map((font) => (
              <option key={font} value={font} style={{ fontFamily: font }}>
                {font}
              </option>
            ))}
          </optgroup>
          <optgroup label={t("resumeBuilder.design.premiumFonts")}>
            {PREMIUM_FONTS.map((font) => (
              <option key={font} value={font} style={{ fontFamily: font }}>
                {font}
              </option>
            ))}
          </optgroup>
        </select>
      </div>

      {/* Spacing */}
      <div className="space-y-2">
        <Label htmlFor="spacing">
          {t("resumeBuilder.design.spacing")}: {resume.spacing}%
        </Label>
        <input
          id="spacing"
          type="range"
          min={50}
          max={150}
          value={resume.spacing}
          onChange={(e) =>
            updateDesign({ spacing: parseInt(e.target.value, 10) })
          }
          className="w-full"
        />
      </div>

      {/* Margins */}
      <div className="space-y-2">
        <Label>{t("resumeBuilder.design.margins")}</Label>
        <div className="grid grid-cols-2 gap-3">
          {(
            [
              "margin_top",
              "margin_bottom",
              "margin_left",
              "margin_right",
            ] as const
          ).map((key) => (
            <div key={key} className="space-y-1">
              <span className="text-xs text-muted-foreground">
                {t(`resumeBuilder.design.${key}`)}
              </span>
              <Input
                type="number"
                min={0}
                max={200}
                value={resume[key]}
                onChange={(e) => {
                  const val = parseInt(e.target.value, 10);
                  if (!isNaN(val) && val >= 0 && val <= 200) {
                    updateDesign({ [key]: val });
                  }
                }}
                className="h-8"
              />
            </div>
          ))}
        </div>
      </div>

      {/* Section Order */}
      <SectionOrderPanel />
    </div>
  );
}
