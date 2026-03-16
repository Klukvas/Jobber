package ai

// ParsedResume represents structured resume data extracted by AI.
type ParsedResume struct {
	FullName       string                `json:"full_name"`
	Email          string                `json:"email"`
	Phone          string                `json:"phone"`
	Location       string                `json:"location"`
	Website        string                `json:"website"`
	LinkedIn       string                `json:"linkedin"`
	GitHub         string                `json:"github"`
	Summary        string                `json:"summary"`
	Experiences    []ParsedExperience    `json:"experiences"`
	Educations     []ParsedEducation     `json:"educations"`
	Skills         []ParsedSkill         `json:"skills"`
	Languages      []ParsedLanguage      `json:"languages"`
	Certifications []ParsedCertification `json:"certifications"`
}

// ParsedExperience represents a work experience entry extracted from a resume.
type ParsedExperience struct {
	Company     string `json:"company"`
	Position    string `json:"position"`
	Location    string `json:"location"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description"`
}

// ParsedEducation represents an education entry extracted from a resume.
type ParsedEducation struct {
	Institution  string `json:"institution"`
	Degree       string `json:"degree"`
	FieldOfStudy string `json:"field_of_study"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	GPA          string `json:"gpa"`
}

// ParsedSkill represents a skill entry extracted from a resume.
type ParsedSkill struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

// ParsedLanguage represents a language entry extracted from a resume.
type ParsedLanguage struct {
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
}

// ParsedCertification represents a certification entry extracted from a resume.
type ParsedCertification struct {
	Name      string `json:"name"`
	Issuer    string `json:"issuer"`
	IssueDate string `json:"issue_date"`
}

const (
	maxResumeTextLength       = 100000
	maxParsedExperiences      = 20
	maxParsedEducations       = 10
	maxParsedSkills           = 50
	maxParsedLanguages        = 10
	maxParsedCertifications   = 20
	maxResumeFieldLength      = 500
	maxDescriptionFieldLength = 5000
)

// validateParsedResume clamps and truncates fields to prevent oversized AI output.
func validateParsedResume(p *ParsedResume) {
	p.FullName = truncateString(p.FullName, maxResumeFieldLength)
	p.Email = truncateString(p.Email, maxResumeFieldLength)
	p.Phone = truncateString(p.Phone, maxResumeFieldLength)
	p.Location = truncateString(p.Location, maxResumeFieldLength)
	p.Website = truncateString(p.Website, maxResumeFieldLength)
	p.LinkedIn = truncateString(p.LinkedIn, maxResumeFieldLength)
	p.GitHub = truncateString(p.GitHub, maxResumeFieldLength)
	p.Summary = truncateString(p.Summary, maxDescriptionFieldLength)

	if len(p.Experiences) > maxParsedExperiences {
		p.Experiences = p.Experiences[:maxParsedExperiences]
	}
	for i := range p.Experiences {
		p.Experiences[i].Company = truncateString(p.Experiences[i].Company, maxResumeFieldLength)
		p.Experiences[i].Position = truncateString(p.Experiences[i].Position, maxResumeFieldLength)
		p.Experiences[i].Location = truncateString(p.Experiences[i].Location, maxResumeFieldLength)
		p.Experiences[i].Description = truncateString(p.Experiences[i].Description, maxDescriptionFieldLength)
	}

	if len(p.Educations) > maxParsedEducations {
		p.Educations = p.Educations[:maxParsedEducations]
	}
	for i := range p.Educations {
		p.Educations[i].Institution = truncateString(p.Educations[i].Institution, maxResumeFieldLength)
		p.Educations[i].Degree = truncateString(p.Educations[i].Degree, maxResumeFieldLength)
		p.Educations[i].FieldOfStudy = truncateString(p.Educations[i].FieldOfStudy, maxResumeFieldLength)
	}

	if len(p.Skills) > maxParsedSkills {
		p.Skills = p.Skills[:maxParsedSkills]
	}

	if len(p.Languages) > maxParsedLanguages {
		p.Languages = p.Languages[:maxParsedLanguages]
	}

	if len(p.Certifications) > maxParsedCertifications {
		p.Certifications = p.Certifications[:maxParsedCertifications]
	}
}
