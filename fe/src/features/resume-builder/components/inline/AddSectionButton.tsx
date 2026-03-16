import { useState, useRef, useEffect } from "react";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { SectionOrderDTO } from "@/shared/types/resume-builder";
import { SECTION_LABEL_KEYS } from "../../constants/sectionLabels";

const EMPTY: readonly never[] = [];

interface AddSectionButtonProps {
  readonly editable?: boolean;
}

export function AddSectionButton({ editable = false }: AddSectionButtonProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const sectionOrder = useResumeBuilderStore(
    (s) => s.resume?.section_order ?? EMPTY,
  );
  const setSectionOrder = useResumeBuilderStore((s) => s.setSectionOrder);

  const hiddenSections = sectionOrder.filter((s) => !s.is_visible);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen]);

  const showSection = (sectionKey: string) => {
    const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
      if (entry.section_key !== sectionKey) return entry;
      return { ...entry, is_visible: true };
    });
    setSectionOrder(updated);
    setIsOpen(false);
  };

  if (!editable || hiddenSections.length === 0) return null;

  return (
    <div ref={menuRef} className="relative mt-4">
      <button
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex w-full items-center justify-center gap-1.5 rounded-lg border-2 border-dashed border-gray-300 py-2.5 text-xs font-medium text-gray-400 transition-colors hover:border-gray-400 hover:text-gray-500"
      >
        <Plus className="h-3.5 w-3.5" />
        {t("resumeBuilder.layout.addSection")}
      </button>

      {isOpen && (
        <div className="absolute bottom-full left-0 right-0 z-10 mb-1 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
          {hiddenSections.map((section) => (
            <button
              key={section.section_key}
              onClick={() => showSection(section.section_key)}
              className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs text-gray-600 transition-colors hover:bg-gray-50"
            >
              <Plus className="h-3 w-3 text-gray-400" />
              {t(
                SECTION_LABEL_KEYS[section.section_key] ?? section.section_key,
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
