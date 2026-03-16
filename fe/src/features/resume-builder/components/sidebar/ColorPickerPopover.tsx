import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { cn } from "@/shared/lib/utils";
import { SidebarPopover } from "./SidebarPopover";
import { COLOR_PALETTES, TEMPLATE_RECOMMENDED_PALETTE } from "./colorPalettes";
import { COLOR_THEMES } from "./colorThemes";
import type { TemplateVariant } from "../preview/shared/templateConfig";

const TEMPLATE_ID_TO_VARIANT: Readonly<Record<string, TemplateVariant>> = {
  "00000000-0000-0000-0000-000000000001": "professional",
  "00000000-0000-0000-0000-000000000002": "modern",
  "00000000-0000-0000-0000-000000000003": "minimal",
  "00000000-0000-0000-0000-000000000004": "executive",
  "00000000-0000-0000-0000-000000000005": "creative",
  "00000000-0000-0000-0000-000000000006": "compact",
  "00000000-0000-0000-0000-000000000007": "elegant",
  "00000000-0000-0000-0000-000000000008": "iconic",
  "00000000-0000-0000-0000-000000000009": "bold",
  "00000000-0000-0000-0000-00000000000a": "accent",
  "00000000-0000-0000-0000-00000000000b": "timeline",
  "00000000-0000-0000-0000-00000000000c": "vivid",
};

const HEX_COLOR_REGEX = /^#[0-9a-fA-F]{6}$/;

interface ColorPickerPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly fullscreen?: boolean;
}

interface ColorPaletteSectionProps {
  readonly label: string;
  readonly selectedColor: string;
  readonly onSelect: (color: string) => void;
  readonly sortedPalettes: typeof COLOR_PALETTES;
  readonly recommendedKey: string;
  readonly t: (key: string) => string;
}

function ColorPaletteSection({
  label,
  selectedColor,
  onSelect,
  sortedPalettes,
  recommendedKey,
  t,
}: ColorPaletteSectionProps) {
  return (
    <div className="space-y-3">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {label}
      </h3>
      {sortedPalettes.map((palette) => (
        <div key={palette.nameKey}>
          <div className="mb-1.5 flex items-center gap-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              {t(palette.nameKey)}
            </span>
            {palette.nameKey === recommendedKey && (
              <span className="rounded bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                {t("resumeBuilder.colors.recommended")}
              </span>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            {palette.colors.map((color) => (
              <button
                key={color}
                onClick={() => onSelect(color)}
                className={cn(
                  "h-7 w-7 rounded-full border-2 transition-transform hover:scale-110",
                  selectedColor === color
                    ? "border-foreground ring-2 ring-primary ring-offset-2"
                    : "border-transparent",
                )}
                style={{ backgroundColor: color }}
                aria-label={color}
              />
            ))}
          </div>
        </div>
      ))}

      {/* Custom color input */}
      <div>
        <span className="mb-1.5 block text-xs font-medium text-muted-foreground">
          {t("resumeBuilder.colors.custom")}
        </span>
        <div className="flex items-center gap-2">
          <input
            type="color"
            value={selectedColor}
            onChange={(e) => onSelect(e.target.value)}
            className="h-8 w-12 cursor-pointer rounded border p-0"
          />
          <input
            value={selectedColor}
            onChange={(e) => {
              const val = e.target.value;
              if (HEX_COLOR_REGEX.test(val)) {
                onSelect(val);
              }
            }}
            className="w-full rounded border border-input bg-background px-2 py-1.5 font-mono text-sm"
            maxLength={7}
          />
        </div>
      </div>
    </div>
  );
}

export function ColorPickerPopover({
  isOpen,
  onClose,
  fullscreen = false,
}: ColorPickerPopoverProps) {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);

  const variant =
    TEMPLATE_ID_TO_VARIANT[resume?.template_id ?? ""] ?? "professional";
  const recommendedKey = TEMPLATE_RECOMMENDED_PALETTE[variant];

  // Sort palettes: recommended first (memoized since recommendedKey rarely changes)
  const sortedPalettes = useMemo(
    () =>
      [...COLOR_PALETTES].sort((a, b) => {
        if (a.nameKey === recommendedKey) return -1;
        if (b.nameKey === recommendedKey) return 1;
        return 0;
      }),
    [recommendedKey],
  );

  return (
    <SidebarPopover
      isOpen={isOpen}
      onClose={onClose}
      title={t("resumeBuilder.design.color")}
      fullscreen={fullscreen}
    >
      <div className="space-y-6">
        {/* Themes grid */}
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {t("resumeBuilder.themes.title")}
          </h3>
          <div className="mt-2 flex flex-wrap gap-2">
            {COLOR_THEMES.map((theme) => {
              const isActive =
                resume?.primary_color === theme.primary &&
                resume?.text_color === theme.text;
              return (
                <button
                  key={theme.nameKey}
                  title={t(theme.nameKey)}
                  onClick={() =>
                    updateDesign({
                      primary_color: theme.primary,
                      text_color: theme.text,
                    })
                  }
                  className={cn(
                    "flex h-8 w-8 items-center justify-center rounded-full border-2 transition-transform hover:scale-110",
                    isActive
                      ? "border-foreground ring-2 ring-primary ring-offset-2"
                      : "border-transparent",
                  )}
                  style={{ backgroundColor: theme.primary }}
                  aria-label={t(theme.nameKey)}
                >
                  <span
                    className="h-3 w-3 rounded-full border border-white/30"
                    style={{ backgroundColor: theme.text }}
                  />
                </button>
              );
            })}
          </div>
        </div>

        <div className="border-t border-border" />

        <ColorPaletteSection
          label={t("resumeBuilder.design.panelColor")}
          selectedColor={resume?.primary_color ?? "#2563eb"}
          onSelect={(color) => updateDesign({ primary_color: color })}
          sortedPalettes={sortedPalettes}
          recommendedKey={recommendedKey}
          t={t}
        />

        <div className="border-t border-border" />

        <ColorPaletteSection
          label={t("resumeBuilder.design.textColor")}
          selectedColor={
            resume?.text_color ?? resume?.primary_color ?? "#2563eb"
          }
          onSelect={(color) => updateDesign({ text_color: color })}
          sortedPalettes={sortedPalettes}
          recommendedKey={recommendedKey}
          t={t}
        />
      </div>
    </SidebarPopover>
  );
}
