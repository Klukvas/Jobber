import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import type { ProjectDTO } from "@/shared/types/resume-builder";

export function ProjectsEditor() {
  const { t } = useTranslation();
  const projects = useResumeBuilderStore((s) => s.resume?.projects ?? []);
  const addProject = useResumeBuilderStore((s) => s.addProject);
  const updateProject = useResumeBuilderStore((s) => s.updateProject);
  const removeProject = useResumeBuilderStore((s) => s.removeProject);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<ProjectDTO>("projects");

  const [expandedId, setExpandedId] = useState<string | null>(null);

  const toggle = (id: string) =>
    setExpandedId((prev) => (prev === id ? null : id));

  const handleAdd = () => {
    const newProject: ProjectDTO = {
      id: crypto.randomUUID(),
      name: "",
      url: "",
      start_date: "",
      end_date: "",
      description: "",
      sort_order: projects.length,
    };
    addProject(newProject);
    persistAdd(newProject);
    setExpandedId(newProject.id);
  };

  const handleRemove = (id: string) => {
    removeProject(id);
    persistRemove(id);
  };

  const handleChange = (
    id: string,
    field: keyof Omit<ProjectDTO, "id" | "sort_order">,
    value: string,
  ) => {
    updateProject(id, { [field]: value });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.projects")}
        </h2>
        <Button variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.projects.add")}
        </Button>
      </div>

      {projects.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.projects.empty")}
        </p>
      )}

      <div className="space-y-3">
        {projects.map((project) => {
          const isExpanded = expandedId === project.id;

          return (
            <div key={project.id} className="rounded-lg border bg-card">
              <button
                type="button"
                onClick={() => toggle(project.id)}
                className="flex w-full items-center justify-between p-4 text-left"
              >
                <span className="font-medium">
                  {project.name || t("resumeBuilder.projects.untitled")}
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
                      <Label htmlFor={`project-name-${project.id}`}>
                        {t("resumeBuilder.projects.name")}
                      </Label>
                      <Input
                        id={`project-name-${project.id}`}
                        value={project.name}
                        onChange={(e) =>
                          handleChange(project.id, "name", e.target.value)
                        }
                        placeholder={t(
                          "resumeBuilder.projects.namePlaceholder",
                        )}
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`project-url-${project.id}`}>
                        {t("resumeBuilder.projects.url")}
                      </Label>
                      <Input
                        id={`project-url-${project.id}`}
                        value={project.url}
                        onChange={(e) =>
                          handleChange(project.id, "url", e.target.value)
                        }
                        placeholder={t("resumeBuilder.projects.urlPlaceholder")}
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`project-start-${project.id}`}>
                        {t("resumeBuilder.projects.startDate")}
                      </Label>
                      <Input
                        id={`project-start-${project.id}`}
                        type="date"
                        value={project.start_date}
                        onChange={(e) =>
                          handleChange(project.id, "start_date", e.target.value)
                        }
                      />
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor={`project-end-${project.id}`}>
                        {t("resumeBuilder.projects.endDate")}
                      </Label>
                      <Input
                        id={`project-end-${project.id}`}
                        type="date"
                        value={project.end_date}
                        onChange={(e) =>
                          handleChange(project.id, "end_date", e.target.value)
                        }
                      />
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <Label htmlFor={`project-desc-${project.id}`}>
                      {t("resumeBuilder.projects.description")}
                    </Label>
                    <Textarea
                      id={`project-desc-${project.id}`}
                      value={project.description}
                      onChange={(e) =>
                        handleChange(project.id, "description", e.target.value)
                      }
                      placeholder={t(
                        "resumeBuilder.projects.descriptionPlaceholder",
                      )}
                      rows={4}
                    />
                  </div>

                  <div className="flex justify-end">
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => handleRemove(project.id)}
                    >
                      <Trash2 className="mr-1 h-4 w-4" />
                      {t("resumeBuilder.projects.remove")}
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
