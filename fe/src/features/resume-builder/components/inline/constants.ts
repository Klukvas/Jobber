import type { ContactDTO } from "@/shared/types/resume-builder";

export const EMPTY_CONTACT: ContactDTO = {
  full_name: "",
  email: "",
  phone: "",
  location: "",
  website: "",
  linkedin: "",
  github: "",
};

export const SKILL_LEVEL_OPTIONS = [
  { value: "beginner", label: "Beginner", labelKey: "resumeBuilder.skillLevels.beginner" },
  { value: "intermediate", label: "Intermediate", labelKey: "resumeBuilder.skillLevels.intermediate" },
  { value: "advanced", label: "Advanced", labelKey: "resumeBuilder.skillLevels.advanced" },
  { value: "expert", label: "Expert", labelKey: "resumeBuilder.skillLevels.expert" },
  { value: "master", label: "Master", labelKey: "resumeBuilder.skillLevels.master" },
] as const;

export const PROFICIENCY_OPTIONS = [
  { value: "elementary", label: "Elementary", labelKey: "resumeBuilder.proficiencyLevels.elementary" },
  { value: "limited_working", label: "Limited Working", labelKey: "resumeBuilder.proficiencyLevels.limited_working" },
  { value: "professional_working", label: "Professional Working", labelKey: "resumeBuilder.proficiencyLevels.professional_working" },
  { value: "full_professional", label: "Full Professional", labelKey: "resumeBuilder.proficiencyLevels.full_professional" },
  { value: "native", label: "Native", labelKey: "resumeBuilder.proficiencyLevels.native" },
] as const;
