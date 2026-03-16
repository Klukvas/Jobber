import { useState, useRef, useEffect } from "react";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type {
  SectionOrderDTO,
  ColumnPlacement,
} from "@/shared/types/resume-builder";
import { SECTION_LABEL_KEYS } from "../../constants/sectionLabels";

const EMPTY: readonly never[] = [];

interface SectionDividerProps {
  /** sort_order value: new section will be inserted at this position */
  readonly insertAtOrder: number;
  readonly editable?: boolean;
  readonly color?: string;
  readonly column?: "main" | "sidebar";
}

export function SectionDivider({
  insertAtOrder,
  editable = false,
  color = "#2b6b4f",
  column,
}: SectionDividerProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
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
        setIsHovered(false);
      }
    };
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen]);

  const addSectionAtPosition = (sectionKey: string) => {
    const targetColumn: ColumnPlacement = column ?? "main";
    const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
      if (entry.section_key === sectionKey) {
        return {
          ...entry,
          is_visible: true,
          sort_order: insertAtOrder,
          column: targetColumn,
        };
      }
      // Only shift sections in the same column
      if (
        entry.sort_order >= insertAtOrder &&
        entry.column === targetColumn &&
        entry.is_visible
      ) {
        return { ...entry, sort_order: entry.sort_order + 1 };
      }
      return entry;
    });
    // Normalize sort_order to be contiguous
    const normalized = [...updated]
      .sort((a, b) => a.sort_order - b.sort_order)
      .map((entry, idx) => ({ ...entry, sort_order: idx }));
    setSectionOrder(normalized);
    setIsOpen(false);
  };

  if (!editable || hiddenSections.length === 0) return null;

  return (
    <div
      ref={menuRef}
      className="relative flex items-center justify-center py-1"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => {
        if (!isOpen) setIsHovered(false);
      }}
    >
      {/* Thin line + circle button */}
      <div
        className={`flex w-full items-center transition-opacity duration-150 ${
          isHovered || isOpen ? "opacity-100" : "opacity-0"
        }`}
      >
        <div className="h-px flex-1 bg-gray-200" />
        <button
          onClick={() => setIsOpen((prev) => !prev)}
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 text-white transition-transform hover:scale-110"
          style={{ borderColor: color, backgroundColor: color }}
          aria-label={t("resumeBuilder.layout.addSection")}
        >
          <Plus className="h-3 w-3" />
        </button>
        <div className="h-px flex-1 bg-gray-200" />
      </div>

      {/* Dropdown of hidden sections */}
      {isOpen && (
        <div className="absolute top-full z-20 mt-1 min-w-[180px] rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
          <div className="px-3 py-1.5 text-[10px] font-medium uppercase tracking-wider text-gray-400">
            {t("resumeBuilder.layout.addSection")}
          </div>
          {hiddenSections.map((section) => (
            <button
              key={section.section_key}
              onClick={() => addSectionAtPosition(section.section_key)}
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
