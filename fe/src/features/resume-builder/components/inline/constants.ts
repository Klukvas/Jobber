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
  { value: "beginner", label: "Beginner" },
  { value: "intermediate", label: "Intermediate" },
  { value: "advanced", label: "Advanced" },
  { value: "expert", label: "Expert" },
  { value: "master", label: "Master" },
] as const;

export const PROFICIENCY_OPTIONS = [
  { value: "elementary", label: "Elementary" },
  { value: "limited_working", label: "Limited Working" },
  { value: "professional_working", label: "Professional Working" },
  { value: "full_professional", label: "Full Professional" },
  { value: "native", label: "Native" },
] as const;
