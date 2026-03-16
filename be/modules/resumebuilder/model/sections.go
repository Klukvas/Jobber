package model

import "time"

// Contact represents resume contact info (1:1 with resume builder).
type Contact struct {
	ID              string
	ResumeBuilderID string
	FullName        string
	Email           string
	Phone           string
	Location        string
	Website         string
	LinkedIn        string
	GitHub          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ContactDTO is the JSON response for contact info.
type ContactDTO struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Website  string `json:"website"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
}

// ToDTO converts Contact to ContactDTO.
func (c *Contact) ToDTO() *ContactDTO {
	return &ContactDTO{
		FullName: c.FullName,
		Email:    c.Email,
		Phone:    c.Phone,
		Location: c.Location,
		Website:  c.Website,
		LinkedIn: c.LinkedIn,
		GitHub:   c.GitHub,
	}
}

// Summary represents resume summary (1:1 with resume builder).
type Summary struct {
	ID              string
	ResumeBuilderID string
	Content         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SummaryDTO is the JSON response for summary.
type SummaryDTO struct {
	Content string `json:"content"`
}

// Experience represents a work experience entry.
type Experience struct {
	ID              string
	ResumeBuilderID string
	Company         string
	Position        string
	Location        string
	StartDate       string
	EndDate         string
	IsCurrent       bool
	Description     string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ExperienceDTO is the JSON response for experience.
type ExperienceDTO struct {
	ID          string `json:"id"`
	Company     string `json:"company"`
	Position    string `json:"position"`
	Location    string `json:"location"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// ToDTO converts Experience to ExperienceDTO.
func (e *Experience) ToDTO() *ExperienceDTO {
	return &ExperienceDTO{
		ID:          e.ID,
		Company:     e.Company,
		Position:    e.Position,
		Location:    e.Location,
		StartDate:   e.StartDate,
		EndDate:     e.EndDate,
		IsCurrent:   e.IsCurrent,
		Description: e.Description,
		SortOrder:   e.SortOrder,
	}
}

// Education represents an education entry.
type Education struct {
	ID              string
	ResumeBuilderID string
	Institution     string
	Degree          string
	FieldOfStudy    string
	StartDate       string
	EndDate         string
	IsCurrent       bool
	GPA             string
	Description     string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EducationDTO is the JSON response for education.
type EducationDTO struct {
	ID           string `json:"id"`
	Institution  string `json:"institution"`
	Degree       string `json:"degree"`
	FieldOfStudy string `json:"field_of_study"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	IsCurrent    bool   `json:"is_current"`
	GPA          string `json:"gpa"`
	Description  string `json:"description"`
	SortOrder    int    `json:"sort_order"`
}

// ToDTO converts Education to EducationDTO.
func (e *Education) ToDTO() *EducationDTO {
	return &EducationDTO{
		ID:           e.ID,
		Institution:  e.Institution,
		Degree:       e.Degree,
		FieldOfStudy: e.FieldOfStudy,
		StartDate:    e.StartDate,
		EndDate:      e.EndDate,
		IsCurrent:    e.IsCurrent,
		GPA:          e.GPA,
		Description:  e.Description,
		SortOrder:    e.SortOrder,
	}
}

// Skill represents a skill entry.
type Skill struct {
	ID              string
	ResumeBuilderID string
	Name            string
	Level           string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SkillDTO is the JSON response for skill.
type SkillDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Level     string `json:"level"`
	SortOrder int    `json:"sort_order"`
}

// ToDTO converts Skill to SkillDTO.
func (s *Skill) ToDTO() *SkillDTO {
	return &SkillDTO{
		ID:        s.ID,
		Name:      s.Name,
		Level:     s.Level,
		SortOrder: s.SortOrder,
	}
}

// Language represents a language entry.
type Language struct {
	ID              string
	ResumeBuilderID string
	Name            string
	Proficiency     string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// LanguageDTO is the JSON response for language.
type LanguageDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
	SortOrder   int    `json:"sort_order"`
}

// ToDTO converts Language to LanguageDTO.
func (l *Language) ToDTO() *LanguageDTO {
	return &LanguageDTO{
		ID:          l.ID,
		Name:        l.Name,
		Proficiency: l.Proficiency,
		SortOrder:   l.SortOrder,
	}
}

// Certification represents a certification entry.
type Certification struct {
	ID              string
	ResumeBuilderID string
	Name            string
	Issuer          string
	IssueDate       string
	ExpiryDate      string
	URL             string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CertificationDTO is the JSON response for certification.
type CertificationDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Issuer     string `json:"issuer"`
	IssueDate  string `json:"issue_date"`
	ExpiryDate string `json:"expiry_date"`
	URL        string `json:"url"`
	SortOrder  int    `json:"sort_order"`
}

// ToDTO converts Certification to CertificationDTO.
func (c *Certification) ToDTO() *CertificationDTO {
	return &CertificationDTO{
		ID:         c.ID,
		Name:       c.Name,
		Issuer:     c.Issuer,
		IssueDate:  c.IssueDate,
		ExpiryDate: c.ExpiryDate,
		URL:        c.URL,
		SortOrder:  c.SortOrder,
	}
}

// Project represents a project entry.
type Project struct {
	ID              string
	ResumeBuilderID string
	Name            string
	URL             string
	StartDate       string
	EndDate         string
	Description     string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProjectDTO is the JSON response for project.
type ProjectDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// ToDTO converts Project to ProjectDTO.
func (p *Project) ToDTO() *ProjectDTO {
	return &ProjectDTO{
		ID:          p.ID,
		Name:        p.Name,
		URL:         p.URL,
		StartDate:   p.StartDate,
		EndDate:     p.EndDate,
		Description: p.Description,
		SortOrder:   p.SortOrder,
	}
}

// Volunteering represents a volunteering entry.
type Volunteering struct {
	ID              string
	ResumeBuilderID string
	Organization    string
	Role            string
	StartDate       string
	EndDate         string
	Description     string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// VolunteeringDTO is the JSON response for volunteering.
type VolunteeringDTO struct {
	ID           string `json:"id"`
	Organization string `json:"organization"`
	Role         string `json:"role"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Description  string `json:"description"`
	SortOrder    int    `json:"sort_order"`
}

// ToDTO converts Volunteering to VolunteeringDTO.
func (v *Volunteering) ToDTO() *VolunteeringDTO {
	return &VolunteeringDTO{
		ID:           v.ID,
		Organization: v.Organization,
		Role:         v.Role,
		StartDate:    v.StartDate,
		EndDate:      v.EndDate,
		Description:  v.Description,
		SortOrder:    v.SortOrder,
	}
}

// CustomSection represents a custom section entry (premium).
type CustomSection struct {
	ID              string
	ResumeBuilderID string
	Title           string
	Content         string
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CustomSectionDTO is the JSON response for custom section.
type CustomSectionDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
}

// ToDTO converts CustomSection to CustomSectionDTO.
func (cs *CustomSection) ToDTO() *CustomSectionDTO {
	return &CustomSectionDTO{
		ID:        cs.ID,
		Title:     cs.Title,
		Content:   cs.Content,
		SortOrder: cs.SortOrder,
	}
}

// SectionOrder represents section ordering and visibility.
type SectionOrder struct {
	ID              string
	ResumeBuilderID string
	SectionKey      string
	SortOrder       int
	IsVisible       bool
	Column          string
}

// SectionOrderDTO is the JSON response for section order.
type SectionOrderDTO struct {
	SectionKey string `json:"section_key"`
	SortOrder  int    `json:"sort_order"`
	IsVisible  bool   `json:"is_visible"`
	Column     string `json:"column"`
}

// ToDTO converts SectionOrder to SectionOrderDTO.
func (so *SectionOrder) ToDTO() *SectionOrderDTO {
	return &SectionOrderDTO{
		SectionKey: so.SectionKey,
		SortOrder:  so.SortOrder,
		IsVisible:  so.IsVisible,
		Column:     so.Column,
	}
}
