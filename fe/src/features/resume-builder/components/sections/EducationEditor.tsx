import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import { Button } from "@/shared/ui/Button";
import { Checkbox } from "@/shared/ui/Checkbox";
import { Plus, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import type { EducationDTO } from "@/shared/types/resume-builder";

function createEmptyEducation(sortOrder: number): EducationDTO {
  return {
    id: crypto.randomUUID(),
    institution: "",
    degree: "",
    field_of_study: "",
    start_date: "",
    end_date: "",
    is_current: false,
    gpa: "",
    description: "",
    sort_order: sortOrder,
  };
}

function EducationCard({
  education,
  onUpdate,
  onRemove,
}: {
  readonly education: EducationDTO;
  readonly onUpdate: (id: string, updates: Partial<EducationDTO>) => void;
  readonly onRemove: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(true);

  const title =
    education.degree || education.institution
      ? `${education.degree}${education.degree && education.institution ? " - " : ""}${education.institution}`
      : t("resumeBuilder.education.newEntry");

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
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor={`edu-institution-${education.id}`}>
                {t("resumeBuilder.education.institution")}
              </Label>
              <Input
                id={`edu-institution-${education.id}`}
                value={education.institution}
                onChange={(e) =>
                  onUpdate(education.id, { institution: e.target.value })
                }
                placeholder={t(
                  "resumeBuilder.education.institutionPlaceholder",
                )}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`edu-degree-${education.id}`}>
                {t("resumeBuilder.education.degree")}
              </Label>
              <Input
                id={`edu-degree-${education.id}`}
                value={education.degree}
                onChange={(e) =>
                  onUpdate(education.id, { degree: e.target.value })
                }
                placeholder={t("resumeBuilder.education.degreePlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`edu-field-${education.id}`}>
                {t("resumeBuilder.education.fieldOfStudy")}
              </Label>
              <Input
                id={`edu-field-${education.id}`}
                value={education.field_of_study}
                onChange={(e) =>
                  onUpdate(education.id, { field_of_study: e.target.value })
                }
                placeholder={t(
                  "resumeBuilder.education.fieldOfStudyPlaceholder",
                )}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`edu-gpa-${education.id}`}>
                {t("resumeBuilder.education.gpa")}
              </Label>
              <Input
                id={`edu-gpa-${education.id}`}
                value={education.gpa}
                onChange={(e) =>
                  onUpdate(education.id, { gpa: e.target.value })
                }
                placeholder={t("resumeBuilder.education.gpaPlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`edu-start-${education.id}`}>
                {t("resumeBuilder.education.startDate")}
              </Label>
              <Input
                id={`edu-start-${education.id}`}
                type="date"
                value={education.start_date}
                onChange={(e) =>
                  onUpdate(education.id, { start_date: e.target.value })
                }
              />
            </div>

            {!education.is_current && (
              <div className="space-y-1.5">
                <Label htmlFor={`edu-end-${education.id}`}>
                  {t("resumeBuilder.education.endDate")}
                </Label>
                <Input
                  id={`edu-end-${education.id}`}
                  type="date"
                  value={education.end_date}
                  onChange={(e) =>
                    onUpdate(education.id, { end_date: e.target.value })
                  }
                />
              </div>
            )}

            <div className="flex items-center gap-2 sm:col-span-2">
              <Checkbox
                id={`edu-current-${education.id}`}
                checked={education.is_current}
                onCheckedChange={(checked) =>
                  onUpdate(education.id, {
                    is_current: checked,
                    end_date: checked ? "" : education.end_date,
                  })
                }
              />
              <Label htmlFor={`edu-current-${education.id}`}>
                {t("resumeBuilder.education.isCurrent")}
              </Label>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={`edu-desc-${education.id}`}>
              {t("resumeBuilder.education.description")}
            </Label>
            <Textarea
              id={`edu-desc-${education.id}`}
              value={education.description}
              onChange={(e) =>
                onUpdate(education.id, { description: e.target.value })
              }
              placeholder={t("resumeBuilder.education.descriptionPlaceholder")}
              rows={4}
            />
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => onRemove(education.id)}
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

export function EducationEditor() {
  const { t } = useTranslation();
  const educations = useResumeBuilderStore((s) => s.resume?.educations ?? []);
  const addEducation = useResumeBuilderStore((s) => s.addEducation);
  const updateEducation = useResumeBuilderStore((s) => s.updateEducation);
  const removeEducation = useResumeBuilderStore((s) => s.removeEducation);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<EducationDTO>("educations");

  const handleAdd = () => {
    const item = createEmptyEducation(educations.length);
    addEducation(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeEducation(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.education")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.actions.add")}
        </Button>
      </div>

      {educations.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.education.empty")}
        </p>
      )}

      <div className="space-y-3">
        {educations.map((edu) => (
          <EducationCard
            key={edu.id}
            education={edu}
            onUpdate={updateEducation}
            onRemove={handleRemove}
          />
        ))}
      </div>
    </div>
  );
}
