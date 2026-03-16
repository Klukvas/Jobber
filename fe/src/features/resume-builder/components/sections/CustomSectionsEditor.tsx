import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import type { CustomSectionDTO } from "@/shared/types/resume-builder";

function createEmptyCustomSection(sortOrder: number): CustomSectionDTO {
  return {
    id: crypto.randomUUID(),
    title: "",
    content: "",
    sort_order: sortOrder,
  };
}

function CustomSectionCard({
  section,
  onUpdate,
  onRemove,
}: {
  readonly section: CustomSectionDTO;
  readonly onUpdate: (id: string, updates: Partial<CustomSectionDTO>) => void;
  readonly onRemove: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(true);

  const title = section.title || t("resumeBuilder.customSections.newEntry");

  return (
    <div className="rounded-lg border bg-card">
      <button
        type="button"
        className="flex w-full items-center justify-between p-4 text-left"
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="font-medium">{title}</span>
        {isOpen ? (
          <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
      </button>

      {isOpen && (
        <div className="space-y-4 border-t px-4 pb-4 pt-4">
          <div className="space-y-1.5">
            <Label htmlFor={`cs-title-${section.id}`}>
              {t("resumeBuilder.customSections.title")}
            </Label>
            <Input
              id={`cs-title-${section.id}`}
              value={section.title}
              onChange={(e) => onUpdate(section.id, { title: e.target.value })}
              placeholder={t("resumeBuilder.customSections.titlePlaceholder")}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={`cs-content-${section.id}`}>
              {t("resumeBuilder.customSections.content")}
            </Label>
            <Textarea
              id={`cs-content-${section.id}`}
              value={section.content}
              onChange={(e) =>
                onUpdate(section.id, { content: e.target.value })
              }
              placeholder={t("resumeBuilder.customSections.contentPlaceholder")}
              rows={4}
            />
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => onRemove(section.id)}
            >
              <Trash2 className="mr-1 h-4 w-4" />
              {t("resumeBuilder.actions.remove")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export function CustomSectionsEditor() {
  const { t } = useTranslation();
  const customSections = useResumeBuilderStore(
    (s) => s.resume?.custom_sections ?? [],
  );
  const addCustomSection = useResumeBuilderStore((s) => s.addCustomSection);
  const updateCustomSection = useResumeBuilderStore(
    (s) => s.updateCustomSection,
  );
  const removeCustomSection = useResumeBuilderStore(
    (s) => s.removeCustomSection,
  );
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<CustomSectionDTO>("custom-sections");

  const handleAdd = () => {
    const item = createEmptyCustomSection(customSections.length);
    addCustomSection(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeCustomSection(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.customSections")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.customSections.add")}
        </Button>
      </div>

      {customSections.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.customSections.empty")}
        </p>
      )}

      <div className="space-y-3">
        {customSections.map((cs) => (
          <CustomSectionCard
            key={cs.id}
            section={cs}
            onUpdate={updateCustomSection}
            onRemove={handleRemove}
          />
        ))}
      </div>
    </div>
  );
}
