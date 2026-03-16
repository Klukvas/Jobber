import { SectionHeader } from "../../inline/SectionHeader";
import { SectionDivider } from "../../inline/SectionDivider";
import { TwoColumnLayout } from "../TwoColumnLayout";
import type { TemplateConfig } from "./templateConfig";
import type { TemplateSetup } from "./useTemplateSetup";
import { useConfigWithOverrides } from "./useConfigWithOverrides";
import {
  SectionIcon,
  SummaryContent,
  ExperienceContent,
  EducationContent,
  SkillsContent,
  LanguagesContent,
  CertificationsContent,
  ProjectsContent,
  VolunteeringContent,
  CustomSectionsContent,
} from "./sectionContent";

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface SectionRendererProps {
  /** The section key (e.g. "summary", "experience", "skills"). */
  readonly sectionKey: string;
  readonly setup: TemplateSetup;
  readonly config: TemplateConfig;
  readonly editable: boolean;
  /** Color override (e.g. white-on-colored sidebar in Modern). */
  readonly sectionColor?: string;
  /** Extra className for inline inputs (e.g. white text in Modern sidebar). */
  readonly inputClassName?: string;
}

interface TemplateLayoutProps {
  readonly setup: TemplateSetup;
  readonly config: TemplateConfig;
  readonly editable: boolean;
  /** Color override passed down to each SectionRenderer in the sidebar. */
  readonly sidebarSectionColor?: string;
  /** inputClassName override for sidebar sections. */
  readonly sidebarInputClassName?: string;
  /** Extra wrapper className for the main column. */
  readonly mainColumnClassName?: string;
  /** Extra wrapper className for the sidebar column. */
  readonly sidebarColumnClassName?: string;
}

// ---------------------------------------------------------------------------
// Section title lookup
// ---------------------------------------------------------------------------

const SECTION_TITLES: Readonly<Record<string, string>> = {
  experience: "Work Experience",
  education: "Education",
  skills: "Skills",
  languages: "Languages",
  certifications: "Certifications",
  projects: "Projects",
  volunteering: "Volunteering",
  custom: "Custom Sections",
  custom_sections: "Custom Sections",
};

const VARIANT_TITLE_OVERRIDES: Readonly<
  Partial<Record<string, Partial<Record<string, string>>>>
> = {
  minimal: { experience: "Experience" },
  modern: { experience: "Experience" },
};

function getSectionTitle(key: string, config: TemplateConfig): string {
  // Summary uses its own title from config
  if (key === "summary") return config.summaryTitle;

  const override = VARIANT_TITLE_OVERRIDES[config.variant]?.[key];
  const base = override ?? SECTION_TITLES[key] ?? key;
  if (!config.sectionTitlePrefix) return base;
  return `${config.sectionTitlePrefix}${base}`;
}

// ---------------------------------------------------------------------------
// Section data lookups
// ---------------------------------------------------------------------------

function getSectionData(key: string, setup: TemplateSetup) {
  switch (key) {
    case "summary":
      return {
        onAdd: undefined,
        isEmpty: false,
        emptyPlaceholder: undefined,
      };
    case "experience":
      return {
        onAdd: setup.experienceSection.handleAdd,
        isEmpty: setup.resume.experiences.length === 0,
        emptyPlaceholder: "Add work experience",
      };
    case "education":
      return {
        onAdd: setup.educationSection.handleAdd,
        isEmpty: setup.resume.educations.length === 0,
        emptyPlaceholder: "Add education",
      };
    case "skills":
      return {
        onAdd: setup.skillsSection.handleAdd,
        isEmpty: setup.resume.skills.length === 0,
        emptyPlaceholder: "Add a skill",
      };
    case "languages":
      return {
        onAdd: setup.languagesSection.handleAdd,
        isEmpty: setup.resume.languages.length === 0,
        emptyPlaceholder: "Add a language",
      };
    case "certifications":
      return {
        onAdd: setup.certificationsSection.handleAdd,
        isEmpty: setup.resume.certifications.length === 0,
        emptyPlaceholder: "Add a certification",
      };
    case "projects":
      return {
        onAdd: setup.projectsSection.handleAdd,
        isEmpty: setup.resume.projects.length === 0,
        emptyPlaceholder: "Add a project",
      };
    case "volunteering":
      return {
        onAdd: setup.volunteeringSection.handleAdd,
        isEmpty: setup.resume.volunteering.length === 0,
        emptyPlaceholder: "Add volunteering",
      };
    case "custom":
    case "custom_sections":
      return {
        onAdd: setup.customSectionsSection.handleAdd,
        isEmpty: setup.resume.custom_sections.length === 0,
        emptyPlaceholder: "Add a custom section",
      };
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// Section content dispatcher
// ---------------------------------------------------------------------------

function renderSectionChildren(
  sectionKey: string,
  setup: TemplateSetup,
  config: TemplateConfig,
  editable: boolean,
  sectionColor?: string,
  inputClass?: string,
) {
  const contentProps = {
    setup,
    config,
    editable,
    sectionColor,
    inputClassName: inputClass,
  };

  switch (sectionKey) {
    case "summary":
      return <SummaryContent {...contentProps} />;
    case "experience":
      return <ExperienceContent {...contentProps} />;
    case "education":
      return <EducationContent {...contentProps} />;
    case "skills":
      return <SkillsContent {...contentProps} />;
    case "languages":
      return <LanguagesContent {...contentProps} />;
    case "certifications":
      return <CertificationsContent {...contentProps} />;
    case "projects":
      return <ProjectsContent {...contentProps} />;
    case "volunteering":
      return <VolunteeringContent {...contentProps} />;
    case "custom":
    case "custom_sections":
      return <CustomSectionsContent {...contentProps} />;
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// SectionRenderer
// ---------------------------------------------------------------------------

/**
 * Renders a single resume section by key, using the shared config/setup.
 * All sections (including summary) use SectionHeader for consistent
 * controls (move up/down, remove) and hover highlight.
 */
export function SectionRenderer({
  sectionKey,
  setup,
  config: rawConfig,
  editable,
  sectionColor,
  inputClassName,
}: SectionRendererProps) {
  const config = useConfigWithOverrides(rawConfig);
  const { color, textColor, hideSection, moveSection, canMoveUp, canMoveDown } =
    setup;
  const effectiveColor = sectionColor ?? color;
  const effectiveTextColor = sectionColor ? sectionColor : textColor;
  const inputClass = inputClassName ?? "";

  // Contact is template-specific (only Modern renders it in the switch)
  if (sectionKey === "contact") {
    return null;
  }

  const data = getSectionData(sectionKey, setup);
  if (!data) return null;

  const title = getSectionTitle(sectionKey, config);
  const iconComponent = config.sectionIcons?.[sectionKey];

  const sectionContent = (
    <SectionHeader
      title={title}
      color={effectiveColor}
      textColor={effectiveTextColor}
      onAdd={data.onAdd}
      isEmpty={data.isEmpty}
      emptyPlaceholder={data.emptyPlaceholder}
      editable={editable}
      onRemoveSection={() => hideSection(sectionKey)}
      onMoveUp={
        canMoveUp(sectionKey) ? () => moveSection(sectionKey, "up") : undefined
      }
      onMoveDown={
        canMoveDown(sectionKey)
          ? () => moveSection(sectionKey, "down")
          : undefined
      }
      variant={config.variant}
    >
      {renderSectionChildren(
        sectionKey,
        setup,
        config,
        editable,
        sectionColor,
        inputClass,
      )}
    </SectionHeader>
  );

  // Iconic: wrap section in a flex row with SectionIcon
  if (iconComponent) {
    return (
      <div key={sectionKey} className="mb-4">
        <div className="flex items-start">
          <SectionIcon icon={iconComponent} color={color} />
          <div className="min-w-0 flex-1">{sectionContent}</div>
        </div>
      </div>
    );
  }

  return sectionContent;
}

// ---------------------------------------------------------------------------
// TemplateLayout
// ---------------------------------------------------------------------------

/**
 * Shared layout component handling TwoColumnLayout vs single-column rendering.
 *
 * Used by most templates except Modern (which has custom sidebar styling
 * and renders its own TwoColumnLayout with sidebarStyle).
 */
export function TemplateLayout({
  setup,
  config,
  editable,
  sidebarSectionColor,
  sidebarInputClassName,
  mainColumnClassName = "pr-4",
  sidebarColumnClassName = "border-l border-gray-200 pl-4",
}: TemplateLayoutProps) {
  const {
    color,
    isTwoColumn,
    layoutMode,
    sidebarWidth,
    visibleSections,
    mainSections,
    sidebarSections,
  } = setup;

  const renderSectionWithDivider = (
    sectionKey: string,
    sortOrder: number,
    column?: "main" | "sidebar",
    overrideSectionColor?: string,
    overrideInputClassName?: string,
  ) => (
    <div key={sectionKey} data-avoid-break>
      <SectionRenderer
        sectionKey={sectionKey}
        setup={setup}
        config={config}
        editable={editable}
        sectionColor={overrideSectionColor}
        inputClassName={overrideInputClassName}
      />
      <SectionDivider
        insertAtOrder={sortOrder + 1}
        editable={editable}
        color={column === "sidebar" && sidebarSectionColor ? "white" : color}
        column={column}
      />
    </div>
  );

  if (isTwoColumn) {
    const effectiveMode =
      layoutMode === "double-right" ? "double-right" : "double-left";

    return (
      <TwoColumnLayout
        sidebarWidth={sidebarWidth}
        layoutMode={effectiveMode}
        mainContent={
          <div className={mainColumnClassName}>
            {mainSections.map((s) =>
              renderSectionWithDivider(s.section_key, s.sort_order, "main"),
            )}
          </div>
        }
        sidebarContent={
          <div className={sidebarColumnClassName}>
            {sidebarSections.map((s) =>
              renderSectionWithDivider(
                s.section_key,
                s.sort_order,
                "sidebar",
                sidebarSectionColor,
                sidebarInputClassName,
              ),
            )}
          </div>
        }
      />
    );
  }

  return (
    <>
      {visibleSections.map((s) =>
        renderSectionWithDivider(s.section_key, s.sort_order),
      )}
    </>
  );
}
