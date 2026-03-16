import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import {
  useExperienceInline,
  useEducationInline,
  useSkillsInline,
  useLanguagesInline,
  useCertificationsInline,
  useProjectsInline,
  useVolunteeringInline,
  useCustomSectionsInline,
} from "../../inline/useInlineSection";
import { useSectionVisibility } from "../../../hooks/useSectionVisibility";
import { useResumePreview } from "../ResumePreviewContext";

/**
 * Extracts ALL shared store selectors, inline-section hooks, visibility helpers,
 * and layout computation that every template repeats verbatim.
 *
 * When wrapped in a `ResumePreviewProvider`, uses the provided FullResumeDTO
 * instead of the global store (used for list-page thumbnails).
 *
 * Returns `null` when the resume is not yet loaded (guards downstream rendering).
 */
export function useTemplateSetup() {
  // ---- Context override (for thumbnails) ----
  const previewResume = useResumePreview();

  // ---- Store selectors (10 hooks) ----
  const storeResume = useResumeBuilderStore((s) => s.resume);
  const updateContact = useResumeBuilderStore((s) => s.updateContact);
  const updateSummary = useResumeBuilderStore((s) => s.updateSummary);
  const updateExperience = useResumeBuilderStore((s) => s.updateExperience);
  const updateEducation = useResumeBuilderStore((s) => s.updateEducation);
  const updateSkill = useResumeBuilderStore((s) => s.updateSkill);
  const updateLanguage = useResumeBuilderStore((s) => s.updateLanguage);
  const updateCertification = useResumeBuilderStore(
    (s) => s.updateCertification,
  );
  const updateProject = useResumeBuilderStore((s) => s.updateProject);
  const updateVolunteering = useResumeBuilderStore((s) => s.updateVolunteering);
  const updateCustomSection = useResumeBuilderStore(
    (s) => s.updateCustomSection,
  );

  // ---- Inline section hooks (8 hooks) ----
  const experienceSection = useExperienceInline();
  const educationSection = useEducationInline();
  const skillsSection = useSkillsInline();
  const languagesSection = useLanguagesInline();
  const certificationsSection = useCertificationsInline();
  const projectsSection = useProjectsInline();
  const volunteeringSection = useVolunteeringInline();
  const customSectionsSection = useCustomSectionsInline();

  // ---- Section visibility ----
  const { hideSection, moveSection, canMoveUp, canMoveDown } =
    useSectionVisibility();

  const resume = previewResume ?? storeResume;
  if (!resume) return null;

  // ---- Derived layout data ----
  const color = resume.primary_color;
  const textColor = resume.text_color ?? resume.primary_color;
  const contact = resume.contact;
  const summary = resume.summary;
  const layoutMode = resume.layout_mode ?? "single";
  const sidebarWidth = resume.sidebar_width ?? 35;

  const visibleSections = resume.section_order
    .filter((s) => s.is_visible)
    .sort((a, b) => a.sort_order - b.sort_order);

  const mainSections = visibleSections.filter((s) => s.column !== "sidebar");
  const sidebarSections = visibleSections.filter((s) => s.column === "sidebar");
  const isTwoColumn =
    layoutMode === "double-left" ||
    layoutMode === "double-right" ||
    layoutMode === "custom";

  return {
    // Core data
    resume,
    color,
    textColor,
    contact,
    summary,

    // Layout
    layoutMode,
    sidebarWidth,
    visibleSections,
    mainSections,
    sidebarSections,
    isTwoColumn,

    // Store updaters
    updateContact,
    updateSummary,
    updateExperience,
    updateEducation,
    updateSkill,
    updateLanguage,
    updateCertification,
    updateProject,
    updateVolunteering,
    updateCustomSection,

    // Inline section handlers
    experienceSection,
    educationSection,
    skillsSection,
    languagesSection,
    certificationsSection,
    projectsSection,
    volunteeringSection,
    customSectionsSection,

    // Visibility controls
    hideSection,
    moveSection,
    canMoveUp,
    canMoveDown,
  } as const;
}

/** The non-null return type, for downstream component props. */
export type TemplateSetup = NonNullable<ReturnType<typeof useTemplateSetup>>;
