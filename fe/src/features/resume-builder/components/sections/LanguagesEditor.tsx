import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Select } from "@/shared/ui/Select";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2 } from "lucide-react";
import type { LanguageDTO } from "@/shared/types/resume-builder";

const PROFICIENCY_LEVELS = [
  "elementary",
  "limited_working",
  "professional_working",
  "full_professional",
  "native",
] as const;

function createEmptyLanguage(sortOrder: number): LanguageDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    proficiency: "",
    sort_order: sortOrder,
  };
}

export function LanguagesEditor() {
  const { t } = useTranslation();
  const languages = useResumeBuilderStore((s) => s.resume?.languages ?? []);
  const addLanguage = useResumeBuilderStore((s) => s.addLanguage);
  const updateLanguage = useResumeBuilderStore((s) => s.updateLanguage);
  const removeLanguage = useResumeBuilderStore((s) => s.removeLanguage);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<LanguageDTO>("languages");

  const handleAdd = () => {
    const item = createEmptyLanguage(languages.length);
    addLanguage(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeLanguage(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.languages")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.actions.add")}
        </Button>
      </div>

      {languages.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.languages.empty")}
        </p>
      )}

      <div className="space-y-3">
        {languages.map((lang) => (
          <div
            key={lang.id}
            className="flex items-end gap-3 rounded-lg border bg-card p-3"
          >
            <div className="flex-1 space-y-1.5">
              <Label htmlFor={`lang-name-${lang.id}`}>
                {t("resumeBuilder.languages.name")}
              </Label>
              <Input
                id={`lang-name-${lang.id}`}
                value={lang.name}
                onChange={(e) =>
                  updateLanguage(lang.id, { name: e.target.value })
                }
                placeholder={t("resumeBuilder.languages.namePlaceholder")}
              />
            </div>

            <div className="w-48 space-y-1.5">
              <Label htmlFor={`lang-proficiency-${lang.id}`}>
                {t("resumeBuilder.languages.proficiency")}
              </Label>
              <Select
                id={`lang-proficiency-${lang.id}`}
                value={lang.proficiency}
                onChange={(e) =>
                  updateLanguage(lang.id, { proficiency: e.target.value })
                }
              >
                <option value="">
                  {t("resumeBuilder.languages.selectProficiency")}
                </option>
                {PROFICIENCY_LEVELS.map((level) => (
                  <option key={level} value={level}>
                    {t(`resumeBuilder.languages.proficiencies.${level}`)}
                  </option>
                ))}
              </Select>
            </div>

            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => handleRemove(lang.id)}
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
