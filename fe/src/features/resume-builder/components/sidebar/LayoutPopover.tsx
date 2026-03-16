import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { SectionOrderPanel } from "../SectionOrderPanel";
import { SidebarPopover } from "./SidebarPopover";
import { LayoutConfigurator } from "./LayoutConfigurator";

interface LayoutPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly fullscreen?: boolean;
}

export function LayoutPopover({
  isOpen,
  onClose,
  fullscreen = false,
}: LayoutPopoverProps) {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);

  return (
    <SidebarPopover
      isOpen={isOpen}
      onClose={onClose}
      title={t("resumeBuilder.sidebar.layout")}
      fullscreen={fullscreen}
    >
      <div className="space-y-5">
        {/* Resume Title */}
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.design.title")}
          </label>
          <input
            value={resume?.title ?? ""}
            onChange={(e) => updateDesign({ title: e.target.value })}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          />
        </div>

        {/* Layout Configurator */}
        <LayoutConfigurator />

        {/* Margins */}
        <div className="space-y-2">
          <label className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.design.margins")}
          </label>
          <div className="grid grid-cols-2 gap-2">
            {(
              [
                "margin_top",
                "margin_bottom",
                "margin_left",
                "margin_right",
              ] as const
            ).map((key) => (
              <div key={key} className="space-y-0.5">
                <span className="text-[10px] text-muted-foreground">
                  {t(`resumeBuilder.design.${key}`)}
                </span>
                <input
                  type="number"
                  min={0}
                  max={200}
                  value={resume?.[key] ?? 40}
                  onChange={(e) => {
                    const val = parseInt(e.target.value, 10);
                    if (!isNaN(val) && val >= 0 && val <= 200) {
                      updateDesign({ [key]: val });
                    }
                  }}
                  className="w-full rounded border border-input bg-background px-2 py-1 text-sm"
                />
              </div>
            ))}
          </div>
        </div>

        {/* Section Order */}
        <SectionOrderPanel />
      </div>
    </SidebarPopover>
  );
}
