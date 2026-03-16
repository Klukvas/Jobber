import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Select } from "@/shared/ui/Select";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2 } from "lucide-react";
import type { SkillDTO } from "@/shared/types/resume-builder";

const SKILL_LEVELS = [
  "beginner",
  "intermediate",
  "advanced",
  "expert",
  "master",
] as const;

function createEmptySkill(sortOrder: number): SkillDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    level: "",
    sort_order: sortOrder,
  };
}

export function SkillsEditor() {
  const { t } = useTranslation();
  const skills = useResumeBuilderStore((s) => s.resume?.skills ?? []);
  const addSkill = useResumeBuilderStore((s) => s.addSkill);
  const updateSkill = useResumeBuilderStore((s) => s.updateSkill);
  const removeSkill = useResumeBuilderStore((s) => s.removeSkill);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<SkillDTO>("skills");

  const handleAdd = () => {
    const item = createEmptySkill(skills.length);
    addSkill(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeSkill(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.skills")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.actions.add")}
        </Button>
      </div>

      {skills.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.skills.empty")}
        </p>
      )}

      <div className="space-y-3">
        {skills.map((skill) => (
          <div
            key={skill.id}
            className="flex items-end gap-3 rounded-lg border bg-card p-3"
          >
            <div className="flex-1 space-y-1.5">
              <Label htmlFor={`skill-name-${skill.id}`}>
                {t("resumeBuilder.skills.name")}
              </Label>
              <Input
                id={`skill-name-${skill.id}`}
                value={skill.name}
                onChange={(e) =>
                  updateSkill(skill.id, { name: e.target.value })
                }
                placeholder={t("resumeBuilder.skills.namePlaceholder")}
              />
            </div>

            <div className="w-40 space-y-1.5">
              <Label htmlFor={`skill-level-${skill.id}`}>
                {t("resumeBuilder.skills.level")}
              </Label>
              <Select
                id={`skill-level-${skill.id}`}
                value={skill.level}
                onChange={(e) =>
                  updateSkill(skill.id, { level: e.target.value })
                }
              >
                <option value="">
                  {t("resumeBuilder.skills.selectLevel")}
                </option>
                {SKILL_LEVELS.map((level) => (
                  <option key={level} value={level}>
                    {t(`resumeBuilder.skills.levels.${level}`)}
                  </option>
                ))}
              </Select>
            </div>

            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => handleRemove(skill.id)}
              aria-label={t("resumeBuilder.actions.remove")}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}
