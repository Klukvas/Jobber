import type { FullResumeDTO } from "@/shared/types/resume-builder";

/** Contact data alone cannot ground a personalized claim about the candidate. */
export function hasCandidateSource(resume: FullResumeDTO): boolean {
  return Boolean(resume.summary?.content.trim()) ||
    resume.experiences.some((entry) => Boolean(entry.description.trim())) ||
    resume.educations.some((entry) => Boolean(entry.institution.trim() || entry.degree.trim())) ||
    resume.skills.some((entry) => Boolean(entry.name.trim())) ||
    resume.languages.some((entry) => Boolean(entry.name.trim())) ||
    resume.certifications.some((entry) => Boolean(entry.name.trim())) ||
    resume.projects.some((entry) => Boolean(entry.name.trim() || entry.description.trim())) ||
    resume.volunteering.some((entry) => Boolean(entry.organization.trim() || entry.description.trim())) ||
    resume.custom_sections.some((entry) => Boolean(entry.content.trim()));
}
