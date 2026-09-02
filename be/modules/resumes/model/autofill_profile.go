package model

// AutofillProfileDTO is the structured form-filling data extracted from an
// Uploaded Resume. Field names mirror the Builder Resume profile subset the
// extension consumes (ext/content/autofill.js), so the extension handles both
// sources with the same code.
type AutofillProfileDTO struct {
	Contact     *AutofillContactDTO     `json:"contact"`
	Summary     *AutofillSummaryDTO     `json:"summary,omitempty"`
	Experiences []AutofillExperienceDTO `json:"experiences"`
	Educations  []AutofillEducationDTO  `json:"educations"`
	Skills      []AutofillSkillDTO      `json:"skills"`
}

// AutofillContactDTO carries the contact fields autofill cares about most.
type AutofillContactDTO struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Website  string `json:"website"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
}

// AutofillSummaryDTO is the professional summary section.
type AutofillSummaryDTO struct {
	Content string `json:"content"`
}

// AutofillExperienceDTO is a work experience entry.
type AutofillExperienceDTO struct {
	Company     string `json:"company"`
	Position    string `json:"position"`
	Location    string `json:"location"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description"`
}

// AutofillEducationDTO is an education entry.
type AutofillEducationDTO struct {
	Institution  string `json:"institution"`
	Degree       string `json:"degree"`
	FieldOfStudy string `json:"field_of_study"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

// AutofillSkillDTO is a skill entry.
type AutofillSkillDTO struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}
