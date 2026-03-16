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
import type { ExperienceDTO } from "@/shared/types/resume-builder";

function createEmptyExperience(sortOrder: number): ExperienceDTO {
  return {
    id: crypto.randomUUID(),
    company: "",
    position: "",
    location: "",
    start_date: "",
    end_date: "",
    is_current: false,
    description: "",
    sort_order: sortOrder,
  };
}

function ExperienceCard({
  experience,
  onUpdate,
  onRemove,
}: {
  readonly experience: ExperienceDTO;
  readonly onUpdate: (id: string, updates: Partial<ExperienceDTO>) => void;
  readonly onRemove: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(true);

  const title =
    experience.position || experience.company
      ? `${experience.position}${experience.position && experience.company ? " - " : ""}${experience.company}`
      : t("resumeBuilder.experience.newEntry");

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
              <Label htmlFor={`exp-company-${experience.id}`}>
                {t("resumeBuilder.experience.company")}
              </Label>
              <Input
                id={`exp-company-${experience.id}`}
                value={experience.company}
                onChange={(e) =>
                  onUpdate(experience.id, { company: e.target.value })
                }
                placeholder={t("resumeBuilder.experience.companyPlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`exp-position-${experience.id}`}>
                {t("resumeBuilder.experience.position")}
              </Label>
              <Input
                id={`exp-position-${experience.id}`}
                value={experience.position}
                onChange={(e) =>
                  onUpdate(experience.id, { position: e.target.value })
                }
                placeholder={t("resumeBuilder.experience.positionPlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`exp-location-${experience.id}`}>
                {t("resumeBuilder.experience.location")}
              </Label>
              <Input
                id={`exp-location-${experience.id}`}
                value={experience.location}
                onChange={(e) =>
                  onUpdate(experience.id, { location: e.target.value })
                }
                placeholder={t("resumeBuilder.experience.locationPlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`exp-start-${experience.id}`}>
                {t("resumeBuilder.experience.startDate")}
              </Label>
              <Input
                id={`exp-start-${experience.id}`}
                type="date"
                value={experience.start_date}
                onChange={(e) =>
                  onUpdate(experience.id, { start_date: e.target.value })
                }
              />
            </div>

            {!experience.is_current && (
              <div className="space-y-1.5">
                <Label htmlFor={`exp-end-${experience.id}`}>
                  {t("resumeBuilder.experience.endDate")}
                </Label>
                <Input
                  id={`exp-end-${experience.id}`}
                  type="date"
                  value={experience.end_date}
                  onChange={(e) =>
                    onUpdate(experience.id, { end_date: e.target.value })
                  }
                />
              </div>
            )}

            <div className="flex items-center gap-2 sm:col-span-2">
              <Checkbox
                id={`exp-current-${experience.id}`}
                checked={experience.is_current}
                onCheckedChange={(checked) =>
                  onUpdate(experience.id, {
                    is_current: checked,
                    end_date: checked ? "" : experience.end_date,
                  })
                }
              />
              <Label htmlFor={`exp-current-${experience.id}`}>
                {t("resumeBuilder.experience.isCurrent")}
              </Label>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={`exp-desc-${experience.id}`}>
              {t("resumeBuilder.experience.description")}
            </Label>
            <Textarea
              id={`exp-desc-${experience.id}`}
              value={experience.description}
              onChange={(e) =>
                onUpdate(experience.id, { description: e.target.value })
              }
              placeholder={t("resumeBuilder.experience.descriptionPlaceholder")}
              rows={4}
            />
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => onRemove(experience.id)}
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

export function ExperienceEditor() {
  const { t } = useTranslation();
  const experiences = useResumeBuilderStore((s) => s.resume?.experiences ?? []);
  const addExperience = useResumeBuilderStore((s) => s.addExperience);
  const updateExperience = useResumeBuilderStore((s) => s.updateExperience);
  const removeExperience = useResumeBuilderStore((s) => s.removeExperience);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<ExperienceDTO>("experiences");

  const handleAdd = () => {
    const item = createEmptyExperience(experiences.length);
    addExperience(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeExperience(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.experience")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.actions.add")}
        </Button>
      </div>

      {experiences.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.experience.empty")}
        </p>
      )}

      <div className="space-y-3">
        {experiences.map((exp) => (
          <ExperienceCard
            key={exp.id}
            experience={exp}
            onUpdate={updateExperience}
            onRemove={handleRemove}
          />
        ))}
      </div>
    </div>
  );
}
