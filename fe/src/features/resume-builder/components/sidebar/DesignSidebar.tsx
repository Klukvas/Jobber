import { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import {
  LayoutTemplate,
  Palette,
  Type,
  SlidersHorizontal,
  LayoutGrid,
  CircleDot,
} from "lucide-react";
import { cn } from "@/shared/lib/utils";
import { Tooltip } from "@/shared/ui/Tooltip";
import { TemplatePickerPopover } from "./TemplatePickerPopover";
import { ColorPickerPopover } from "./ColorPickerPopover";
import { TypographyPopover } from "./TypographyPopover";
import { LayoutPopover } from "./LayoutPopover";
import { SectionsPopover } from "./SectionsPopover";
import { SkillDisplayPopover } from "./SkillDisplayPopover";

type PopoverType =
  | "template"
  | "color"
  | "typography"
  | "layout"
  | "sections"
  | "skillDisplay"
  | null;

const SIDEBAR_ITEMS: {
  key: PopoverType;
  icon: typeof LayoutTemplate;
  tooltipKey: string;
}[] = [
  {
    key: "template",
    icon: LayoutTemplate,
    tooltipKey: "resumeBuilder.sidebar.templates",
  },
  { key: "color", icon: Palette, tooltipKey: "resumeBuilder.sidebar.colors" },
  {
    key: "typography",
    icon: Type,
    tooltipKey: "resumeBuilder.sidebar.typography",
  },
  {
    key: "layout",
    icon: SlidersHorizontal,
    tooltipKey: "resumeBuilder.sidebar.layout",
  },
  {
    key: "sections",
    icon: LayoutGrid,
    tooltipKey: "resumeBuilder.sidebar.sections",
  },
  {
    key: "skillDisplay",
    icon: CircleDot,
    tooltipKey: "resumeBuilder.sidebar.skillDisplay",
  },
];

interface DesignSidebarProps {
  /** "vertical" for desktop sidebar, "grid" for mobile sheet */
  readonly layout?: "vertical" | "grid";
}

export function DesignSidebar({ layout = "vertical" }: DesignSidebarProps) {
  const { t } = useTranslation();
  const [activePopover, setActivePopover] = useState<PopoverType>(null);

  const togglePopover = useCallback((key: PopoverType) => {
    setActivePopover((current) => (current === key ? null : key));
  }, []);

  const closePopover = useCallback(() => {
    setActivePopover(null);
  }, []);

  if (layout === "grid") {
    return (
      <div>
        <div className="grid grid-cols-3 gap-3">
          {SIDEBAR_ITEMS.map(({ key, icon: Icon, tooltipKey }) => (
            <button
              key={key}
              onClick={() => togglePopover(key)}
              className={cn(
                "flex flex-col items-center gap-2 rounded-lg border p-4 transition-colors",
                activePopover === key
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
            >
              <Icon className="h-6 w-6" />
              <span className="text-xs font-medium">{t(tooltipKey)}</span>
            </button>
          ))}
        </div>

        <TemplatePickerPopover
          isOpen={activePopover === "template"}
          onClose={closePopover}
        />
        <ColorPickerPopover
          isOpen={activePopover === "color"}
          onClose={closePopover}
          fullscreen
        />
        <TypographyPopover
          isOpen={activePopover === "typography"}
          onClose={closePopover}
          fullscreen
        />
        <LayoutPopover
          isOpen={activePopover === "layout"}
          onClose={closePopover}
          fullscreen
        />
        <SectionsPopover
          isOpen={activePopover === "sections"}
          onClose={closePopover}
          fullscreen
        />
        <SkillDisplayPopover
          isOpen={activePopover === "skillDisplay"}
          onClose={closePopover}
          fullscreen
        />
      </div>
    );
  }

  return (
    <div className="relative flex h-full">
      {/* Icon bar */}
      <div className="flex w-14 flex-col items-center gap-1 border-r bg-muted/80 py-3">
        {SIDEBAR_ITEMS.map(({ key, icon: Icon, tooltipKey }) => (
          <Tooltip key={key} content={t(tooltipKey)} side="right">
            <button
              onClick={() => togglePopover(key)}
              className={cn(
                "flex h-10 w-10 items-center justify-center rounded-lg transition-colors",
                activePopover === key
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
              aria-label={t(tooltipKey)}
            >
              <Icon className="h-5 w-5" />
            </button>
          </Tooltip>
        ))}
      </div>

      {/* Popovers */}
      <TemplatePickerPopover
        isOpen={activePopover === "template"}
        onClose={closePopover}
      />
      <ColorPickerPopover
        isOpen={activePopover === "color"}
        onClose={closePopover}
      />
      <TypographyPopover
        isOpen={activePopover === "typography"}
        onClose={closePopover}
      />
      <LayoutPopover
        isOpen={activePopover === "layout"}
        onClose={closePopover}
      />
      <SectionsPopover
        isOpen={activePopover === "sections"}
        onClose={closePopover}
      />
      <SkillDisplayPopover
        isOpen={activePopover === "skillDisplay"}
        onClose={closePopover}
      />
    </div>
  );
}
