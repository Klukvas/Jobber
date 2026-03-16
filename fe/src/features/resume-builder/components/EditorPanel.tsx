import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { SectionKey } from "@/shared/types/resume-builder";
import {
  User,
  FileText,
  Briefcase,
  GraduationCap,
  Wrench,
  Languages,
  Award,
  FolderOpen,
  Heart,
  List,
  LayoutList,
  Palette,
} from "lucide-react";
import { cn } from "@/shared/lib/utils";
import { ContactEditor } from "./sections/ContactEditor";
import { SummaryEditor } from "./sections/SummaryEditor";
import { ExperienceEditor } from "./sections/ExperienceEditor";
import { EducationEditor } from "./sections/EducationEditor";
import { SkillsEditor } from "./sections/SkillsEditor";
import { LanguagesEditor } from "./sections/LanguagesEditor";
import { CertificationsEditor } from "./sections/CertificationsEditor";
import { ProjectsEditor } from "./sections/ProjectsEditor";
import { VolunteeringEditor } from "./sections/VolunteeringEditor";
import { CustomSectionsEditor } from "./sections/CustomSectionsEditor";
import { DesignPanel } from "./DesignPanel";

interface SectionNavItem {
  key: SectionKey | "design";
  icon: React.ElementType;
  labelKey: string;
}

const sectionNav: SectionNavItem[] = [
  { key: "contact", icon: User, labelKey: "resumeBuilder.sections.contact" },
  {
    key: "summary",
    icon: FileText,
    labelKey: "resumeBuilder.sections.summary",
  },
  {
    key: "experience",
    icon: Briefcase,
    labelKey: "resumeBuilder.sections.experience",
  },
  {
    key: "education",
    icon: GraduationCap,
    labelKey: "resumeBuilder.sections.education",
  },
  { key: "skills", icon: Wrench, labelKey: "resumeBuilder.sections.skills" },
  {
    key: "languages",
    icon: Languages,
    labelKey: "resumeBuilder.sections.languages",
  },
  {
    key: "certifications",
    icon: Award,
    labelKey: "resumeBuilder.sections.certifications",
  },
  {
    key: "projects",
    icon: FolderOpen,
    labelKey: "resumeBuilder.sections.projects",
  },
  {
    key: "volunteering",
    icon: Heart,
    labelKey: "resumeBuilder.sections.volunteering",
  },
  {
    key: "custom_sections",
    icon: List,
    labelKey: "resumeBuilder.sections.customSections",
  },
];

const designNav: SectionNavItem = {
  key: "design" as SectionKey,
  icon: Palette,
  labelKey: "resumeBuilder.sections.design",
};

export function EditorPanel() {
  const { t } = useTranslation();
  const activeSection = useResumeBuilderStore((s) => s.activeSection);
  const setActiveSection = useResumeBuilderStore((s) => s.setActiveSection);

  return (
    <div className="flex flex-col">
      {/* Section navigation */}
      <div className="flex gap-1 overflow-x-auto border-b p-2">
        {sectionNav.map((item) => {
          const Icon = item.icon;
          const isActive = activeSection === item.key;
          return (
            <button
              key={item.key}
              onClick={() => setActiveSection(item.key as SectionKey)}
              className={cn(
                "flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors",
                isActive
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
            >
              <Icon className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">{t(item.labelKey)}</span>
            </button>
          );
        })}
        <button
          onClick={() => setActiveSection("design" as SectionKey)}
          className={cn(
            "flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-sm transition-colors",
            activeSection === ("design" as SectionKey)
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:bg-muted hover:text-foreground",
          )}
        >
          <LayoutList className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">{t(designNav.labelKey)}</span>
        </button>
      </div>

      {/* Active section editor */}
      <div className="p-4">
        {activeSection === "contact" && <ContactEditor />}
        {activeSection === "summary" && <SummaryEditor />}
        {activeSection === "experience" && <ExperienceEditor />}
        {activeSection === "education" && <EducationEditor />}
        {activeSection === "skills" && <SkillsEditor />}
        {activeSection === "languages" && <LanguagesEditor />}
        {activeSection === "certifications" && <CertificationsEditor />}
        {activeSection === "projects" && <ProjectsEditor />}
        {activeSection === "volunteering" && <VolunteeringEditor />}
        {activeSection === "custom_sections" && <CustomSectionsEditor />}
        {activeSection === ("design" as SectionKey) && <DesignPanel />}
      </div>
    </div>
  );
}
