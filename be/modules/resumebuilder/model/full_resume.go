package model

// FullResumeDTO aggregates all resume sections into a single response.
type FullResumeDTO struct {
	*ResumeBuilderDTO
	Contact        *ContactDTO        `json:"contact"`
	Summary        *SummaryDTO        `json:"summary"`
	Experiences    []*ExperienceDTO   `json:"experiences"`
	Educations     []*EducationDTO    `json:"educations"`
	Skills         []*SkillDTO        `json:"skills"`
	Languages      []*LanguageDTO     `json:"languages"`
	Certifications []*CertificationDTO `json:"certifications"`
	Projects       []*ProjectDTO      `json:"projects"`
	Volunteering   []*VolunteeringDTO `json:"volunteering"`
	CustomSections []*CustomSectionDTO `json:"custom_sections"`
	SectionOrder   []*SectionOrderDTO `json:"section_order"`
}
