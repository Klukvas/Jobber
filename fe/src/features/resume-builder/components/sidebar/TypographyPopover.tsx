import { useId } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { SidebarPopover } from "./SidebarPopover";

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

interface TypographyPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly fullscreen?: boolean;
}

export function TypographyPopover({
  isOpen,
  onClose,
  fullscreen = false,
}: TypographyPopoverProps) {
  const prefix = useId();
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);

  return (
    <SidebarPopover
      isOpen={isOpen}
      onClose={onClose}
      title={t("resumeBuilder.design.font")}
      fullscreen={fullscreen}
    >
      <div className="space-y-5">
        {/* Font Family */}
        <div className="space-y-2">
          <label htmlFor={`${prefix}-font`} className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.design.font")}
          </label>
          <select
            id={`${prefix}-font`}
            value={resume?.font_family ?? "Georgia"}
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

        {/* Font Size */}
        <div className="space-y-2">
          <label htmlFor={`${prefix}-size`} className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.design.fontSize")}: {resume?.font_size ?? 12}px
          </label>
          <input
            type="range"
            id={`${prefix}-size`}
            aria-valuetext={`${resume?.font_size ?? 12}px`}
            min={8}
            max={18}
            value={resume?.font_size ?? 12}
            onChange={(e) =>
              updateDesign({ font_size: parseInt(e.target.value, 10) })
            }
            className="w-full accent-primary"
          />
        </div>

        {/* Line Spacing */}
        <div className="space-y-2">
          <label htmlFor={`${prefix}-spacing`} className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.design.spacing")}: {resume?.spacing ?? 100}%
          </label>
          <input
            type="range"
            id={`${prefix}-spacing`}
            aria-valuetext={`${resume?.spacing ?? 100}%`}
            min={50}
            max={150}
            value={resume?.spacing ?? 100}
            onChange={(e) =>
              updateDesign({ spacing: parseInt(e.target.value, 10) })
            }
            className="w-full accent-primary"
          />
        </div>
      </div>
    </SidebarPopover>
  );
}
