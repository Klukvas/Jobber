import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { Button } from "@/shared/ui/Button";
import { ArrowUp, ArrowDown, Eye, EyeOff } from "lucide-react";
import type { SectionOrderDTO } from "@/shared/types/resume-builder";
import { SECTION_LABEL_KEYS } from "../constants/sectionLabels";

const EMPTY: readonly never[] = [];

export function SectionOrderPanel() {
  const { t } = useTranslation();
  const sectionOrder = useResumeBuilderStore(
    (s) => s.resume?.section_order ?? EMPTY,
  );
  const setSectionOrder = useResumeBuilderStore((s) => s.setSectionOrder);

  const sorted = [...sectionOrder].sort((a, b) => a.sort_order - b.sort_order);

  const moveUp = (index: number) => {
    if (index <= 0) return;
    const updated: SectionOrderDTO[] = sorted.map((item, i) => {
      if (i === index) return { ...item, sort_order: item.sort_order - 1 };
      if (i === index - 1) return { ...item, sort_order: item.sort_order + 1 };
      return item;
    });
    setSectionOrder(updated);
  };

  const moveDown = (index: number) => {
    if (index >= sorted.length - 1) return;
    const updated: SectionOrderDTO[] = sorted.map((item, i) => {
      if (i === index) return { ...item, sort_order: item.sort_order + 1 };
      if (i === index + 1) return { ...item, sort_order: item.sort_order - 1 };
      return item;
    });
    setSectionOrder(updated);
  };

  const toggleVisibility = (index: number) => {
    const updated: SectionOrderDTO[] = sorted.map((item, i) => {
      if (i === index) return { ...item, is_visible: !item.is_visible };
      return item;
    });
    setSectionOrder(updated);
  };

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-muted-foreground">
        {t("resumeBuilder.design.sectionOrder")}
      </h3>
      <div className="space-y-1">
        {sorted.map((section, index) => (
          <div
            key={section.section_key}
            className="flex items-center gap-2 rounded-md border px-3 py-2"
          >
            <span
              className={`flex-1 text-sm ${!section.is_visible ? "text-muted-foreground line-through" : ""}`}
            >
              {t(
                SECTION_LABEL_KEYS[section.section_key] ?? section.section_key,
              )}
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              disabled={index === 0}
              onClick={() => moveUp(index)}
              aria-label="Move up"
            >
              <ArrowUp className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              disabled={index === sorted.length - 1}
              onClick={() => moveDown(index)}
              aria-label="Move down"
            >
              <ArrowDown className="h-3.5 w-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => toggleVisibility(index)}
              aria-label={section.is_visible ? "Hide section" : "Show section"}
            >
              {section.is_visible ? (
                <Eye className="h-3.5 w-3.5" />
              ) : (
                <EyeOff className="h-3.5 w-3.5 text-muted-foreground" />
              )}
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}
