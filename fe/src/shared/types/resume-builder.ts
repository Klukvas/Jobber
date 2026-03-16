// Resume Builder types

export type LayoutMode = "single" | "double-left" | "double-right" | "custom";
export type ColumnPlacement = "main" | "sidebar";
export type SkillDisplayMode =
  | ""
  | "text-level"
  | "pill"
  | "grid-level"
  | "vertical"
  | "text-only"
  | "dots"
  | "bar"
  | "square"
  | "star"
  | "circle"
  | "segmented"
  | "bubble";

export interface ResumeBuilderDTO {
  id: string;
  title: string;
  template_id: string;
  font_family: string;
  primary_color: string;
  text_color: string;
  spacing: number;
  margin_top: number;
  margin_bottom: number;
  margin_left: number;
  margin_right: number;
  layout_mode: LayoutMode;
  sidebar_width: number;
  font_size: number;
  skill_display: SkillDisplayMode;
  created_at: string;
  updated_at: string;
}

export interface ContactDTO {
  full_name: string;
  email: string;
  phone: string;
  location: string;
  website: string;
  linkedin: string;
  github: string;
}

export interface SummaryDTO {
  content: string;
}

export interface ExperienceDTO {
  id: string;
  company: string;
  position: string;
  location: string;
  start_date: string;
  end_date: string;
  is_current: boolean;
  description: string;
  sort_order: number;
}

export interface EducationDTO {
  id: string;
  institution: string;
  degree: string;
  field_of_study: string;
  start_date: string;
  end_date: string;
  is_current: boolean;
  gpa: string;
  description: string;
  sort_order: number;
}

export interface SkillDTO {
  id: string;
  name: string;
  level: string;
  sort_order: number;
}

export interface LanguageDTO {
  id: string;
  name: string;
  proficiency: string;
  sort_order: number;
}

export interface CertificationDTO {
  id: string;
  name: string;
  issuer: string;
  issue_date: string;
  expiry_date: string;
  url: string;
  sort_order: number;
}

export interface ProjectDTO {
  id: string;
  name: string;
  url: string;
  start_date: string;
  end_date: string;
  description: string;
  sort_order: number;
}

export interface VolunteeringDTO {
  id: string;
  organization: string;
  role: string;
  start_date: string;
  end_date: string;
  description: string;
  sort_order: number;
}

export interface CustomSectionDTO {
  id: string;
  title: string;
  content: string;
  sort_order: number;
}

export interface SectionOrderDTO {
  section_key: string;
  sort_order: number;
  is_visible: boolean;
  column: ColumnPlacement;
}

export interface FullResumeDTO extends ResumeBuilderDTO {
  contact: ContactDTO | null;
  summary: SummaryDTO | null;
  experiences: ExperienceDTO[];
  educations: EducationDTO[];
  skills: SkillDTO[];
  languages: LanguageDTO[];
  certifications: CertificationDTO[];
  projects: ProjectDTO[];
  volunteering: VolunteeringDTO[];
  custom_sections: CustomSectionDTO[];
  section_order: SectionOrderDTO[];
}

// Request types
export interface CreateResumeBuilderRequest {
  title?: string;
  template_id?: string;
}

export interface UpdateResumeBuilderRequest {
  title?: string;
  template_id?: string;
  font_family?: string;
  primary_color?: string;
  text_color?: string;
  spacing?: number;
  margin_top?: number;
  margin_bottom?: number;
  margin_left?: number;
  margin_right?: number;
  layout_mode?: LayoutMode;
  sidebar_width?: number;
  font_size?: number;
  skill_display?: SkillDisplayMode;
}

export interface UpsertContactRequest {
  full_name: string;
  email: string;
  phone: string;
  location: string;
  website: string;
  linkedin: string;
  github: string;
}

export interface UpsertSummaryRequest {
  content: string;
}

export interface BatchUpdateSectionOrderRequest {
  sections: SectionOrderDTO[];
}

// Section key type
export type SectionKey =
  | "contact"
  | "summary"
  | "experience"
  | "education"
  | "skills"
  | "languages"
  | "certifications"
  | "projects"
  | "volunteering"
  | "custom_sections";

// Template info
export interface ResumeTemplateInfo {
  id: string;
  name: string;
  displayName: string;
  description: string;
  isPremium: boolean;
}
