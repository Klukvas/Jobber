import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import type { VolunteeringDTO } from "@/shared/types/resume-builder";

export function VolunteeringEditor() {
  const { t } = useTranslation();
  const volunteering = useResumeBuilderStore(
    (s) => s.resume?.volunteering ?? [],
  );
  const addVolunteering = useResumeBuilderStore((s) => s.addVolunteering);
  const updateVolunteering = useResumeBuilderStore((s) => s.updateVolunteering);
  const removeVolunteering = useResumeBuilderStore((s) => s.removeVolunteering);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<VolunteeringDTO>("volunteering");

  const [expandedId, setExpandedId] = useState<string | null>(null);

  const toggle = (id: string) =>
    setExpandedId((prev) => (prev === id ? null : id));

  const handleAdd = () => {
    const newEntry: VolunteeringDTO = {
      id: crypto.randomUUID(),
      organization: "",
      role: "",
      start_date: "",
      end_date: "",
      description: "",
      sort_order: volunteering.length,
    };
    addVolunteering(newEntry);
    persistAdd(newEntry);
    setExpandedId(newEntry.id);
  };

  const handleRemove = (id: string) => {
    removeVolunteering(id);
    persistRemove(id);
  };

  const handleChange = (
    id: string,
    field: keyof Omit<VolunteeringDTO, "id" | "sort_order">,
    value: string,
  ) => {
    updateVolunteering(id, { [field]: value });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.volunteering")}
        </h2>
        <Button variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.volunteering.add")}
        </Button>
      </div>

      {volunteering.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.volunteering.empty")}
        </p>
      )}

      <div className="space-y-3">
        {volunteering.map((entry) => {
          const isExpanded = expandedId === entry.id;

          return (
            <div key={entry.id} className="rounded-lg border bg-card">
              <button
                type="button"
                onClick={() => toggle(entry.id)}
                className="flex w-full items-center justify-between p-4 text-left"
              >
                <span className="font-medium">
                  {entry.organization ||
                    t("resumeBuilder.volunteering.untitled")}
                </span>
                {isExpanded ? (
                  <ChevronUp className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                )}
              </button>

              {isExpanded && (
                <div className="space-y-4 border-t px-4 pb-4 pt-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label htmlFor={`vol-org-${entry.id}`}>
                        {t("resumeBuilder.volunteering.organization")}
                      </Label>
                      <Input
                        id={`vol-org-${entry.id}`}
                        value={entry.organization}
                        onChange={(e) =>
                          handleChange(entry.id, "organization", e.target.value)
                        }
                        placeholder={t(
                          "resumeBuilder.volunteering.organizationPlaceholder",
                        )}
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`vol-role-${entry.id}`}>
                        {t("resumeBuilder.volunteering.role")}
                      </Label>
                      <Input
                        id={`vol-role-${entry.id}`}
                        value={entry.role}
                        onChange={(e) =>
                          handleChange(entry.id, "role", e.target.value)
                        }
                        placeholder={t(
                          "resumeBuilder.volunteering.rolePlaceholder",
                        )}
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`vol-start-${entry.id}`}>
                        {t("resumeBuilder.volunteering.startDate")}
                      </Label>
                      <Input
                        id={`vol-start-${entry.id}`}
                        type="date"
                        value={entry.start_date}
                        onChange={(e) =>
                          handleChange(entry.id, "start_date", e.target.value)
                        }
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`vol-end-${entry.id}`}>
                        {t("resumeBuilder.volunteering.endDate")}
                      </Label>
                      <Input
                        id={`vol-end-${entry.id}`}
                        type="date"
                        value={entry.end_date}
                        onChange={(e) =>
                          handleChange(entry.id, "end_date", e.target.value)
                        }
                      />
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor={`vol-desc-${entry.id}`}>
                      {t("resumeBuilder.volunteering.description")}
                    </Label>
                    <Textarea
                      id={`vol-desc-${entry.id}`}
                      value={entry.description}
                      onChange={(e) =>
                        handleChange(entry.id, "description", e.target.value)
                      }
                      placeholder={t(
                        "resumeBuilder.volunteering.descriptionPlaceholder",
                      )}
                      rows={4}
                    />
                  </div>

                  <div className="flex justify-end">
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRemove(entry.id)}
                    >
                      <Trash2 className="mr-1 h-4 w-4" />
                      {t("resumeBuilder.volunteering.remove")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
