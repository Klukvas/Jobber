import type { FullResumeDTO } from "@/shared/types/resume-builder";

export const filledResumeFixture: FullResumeDTO = {
  id: "template-field-regression", title: "Synthetic template QA", template_id: "modern",
  font_family: "Inter", primary_color: "#2563eb", text_color: "#2563eb",
  spacing: 115, margin_top: 20, margin_bottom: 20, margin_left: 20, margin_right: 20,
  layout_mode: "single", sidebar_width: 35, font_size: 12, skill_display: "",
  created_at: "2026-10-24", updated_at: "2026-10-24",
  contact: { full_name: "QA Mobile Resume", email: "qa@example.com", phone: "+123456789", location: "QA City", website: "qa.example.com", linkedin: "linkedin.com/in/qa", github: "github.com/qa" },
  summary: { content: "QA summary first line\nQA summary second line" },
  experiences: [{ id: "exp", position: "QA Engineer", company: "QA Company", location: "QA Office", start_date: "2025-03", end_date: "", is_current: true, description: "QA work first line\nQA work second line", sort_order: 0 }],
  educations: [{ id: "edu", institution: "QA University", degree: "QA Degree", field_of_study: "QA Studies", start_date: "2020-09", end_date: "2024-06", is_current: false, gpa: "3.9", description: "", sort_order: 0 }],
  skills: [{ id: "skill", name: "QA TypeScript", level: "expert", sort_order: 0 }],
  languages: [{ id: "lang", name: "QA English", proficiency: "full_professional", sort_order: 0 }],
  certifications: [{ id: "cert", name: "QA Certificate", issuer: "QA Issuer", issue_date: "2024-05-06", expiry_date: "", url: "", sort_order: 0 }],
  projects: [{ id: "project", name: "QA Project", url: "https://example.com/project", start_date: "2023-01", end_date: "2023-12", description: "QA project first line\nQA project second line", sort_order: 0 }],
  volunteering: [{ id: "vol", role: "QA Volunteer", organization: "QA Organization", start_date: "2022-01", end_date: "2022-12", description: "QA volunteer first line\nQA volunteer second line", sort_order: 0 }],
  custom_sections: [{ id: "custom", title: "QA Custom", content: "QA custom first line\nQA custom second line", sort_order: 0 }],
  section_order: ["contact", "summary", "experience", "education", "skills", "languages", "certifications", "projects", "volunteering", "custom"].map((section_key, sort_order) => ({ section_key, sort_order, is_visible: true, column: "main" })),
};
