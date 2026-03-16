package model

// CreateResumeBuilderRequest is the request payload to create a resume builder.
type CreateResumeBuilderRequest struct {
	Title      string `json:"title" binding:"max=255"`
	TemplateID string `json:"template_id" binding:"max=100"`
}

// UpdateResumeBuilderRequest is the request payload to update resume builder metadata.
type UpdateResumeBuilderRequest struct {
	Title        *string `json:"title,omitempty" binding:"omitempty,max=255"`
	TemplateID   *string `json:"template_id,omitempty" binding:"omitempty,max=100"`
	FontFamily   *string `json:"font_family,omitempty" binding:"omitempty,max=50"`
	PrimaryColor *string `json:"primary_color,omitempty" binding:"omitempty,max=20"`
	TextColor    *string `json:"text_color,omitempty" binding:"omitempty,max=20"`
	Spacing      *int    `json:"spacing,omitempty"`
	MarginTop    *int    `json:"margin_top,omitempty"`
	MarginBottom *int    `json:"margin_bottom,omitempty"`
	MarginLeft   *int    `json:"margin_left,omitempty"`
	MarginRight  *int    `json:"margin_right,omitempty"`
	LayoutMode   *string `json:"layout_mode,omitempty" binding:"omitempty,max=20"`
	SidebarWidth *int    `json:"sidebar_width,omitempty"`
	FontSize     *int    `json:"font_size,omitempty"`
	SkillDisplay *string `json:"skill_display,omitempty" binding:"omitempty,max=20"`
}

// UpsertContactRequest is the request payload to upsert contact info.
type UpsertContactRequest struct {
	FullName string `json:"full_name" binding:"max=255"`
	Email    string `json:"email" binding:"max=255"`
	Phone    string `json:"phone" binding:"max=50"`
	Location string `json:"location" binding:"max=255"`
	Website  string `json:"website" binding:"max=500"`
	LinkedIn string `json:"linkedin" binding:"max=500"`
	GitHub   string `json:"github" binding:"max=500"`
}

// UpsertSummaryRequest is the request payload to upsert summary.
type UpsertSummaryRequest struct {
	Content string `json:"content" binding:"max=5000"`
}

// CreateExperienceRequest is the request payload to add an experience.
type CreateExperienceRequest struct {
	Company     string `json:"company" binding:"max=255"`
	Position    string `json:"position" binding:"max=255"`
	Location    string `json:"location" binding:"max=255"`
	StartDate   string `json:"start_date" binding:"max=20"`
	EndDate     string `json:"end_date" binding:"max=20"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description" binding:"max=10000"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateExperienceRequest is the request payload to update an experience.
type UpdateExperienceRequest struct {
	Company     *string `json:"company,omitempty" binding:"omitempty,max=255"`
	Position    *string `json:"position,omitempty" binding:"omitempty,max=255"`
	Location    *string `json:"location,omitempty" binding:"omitempty,max=255"`
	StartDate   *string `json:"start_date,omitempty" binding:"omitempty,max=20"`
	EndDate     *string `json:"end_date,omitempty" binding:"omitempty,max=20"`
	IsCurrent   *bool   `json:"is_current,omitempty"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=10000"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// CreateEducationRequest is the request payload to add an education.
type CreateEducationRequest struct {
	Institution  string `json:"institution" binding:"max=255"`
	Degree       string `json:"degree" binding:"max=255"`
	FieldOfStudy string `json:"field_of_study" binding:"max=255"`
	StartDate    string `json:"start_date" binding:"max=20"`
	EndDate      string `json:"end_date" binding:"max=20"`
	IsCurrent    bool   `json:"is_current"`
	GPA          string `json:"gpa" binding:"max=20"`
	Description  string `json:"description" binding:"max=10000"`
	SortOrder    int    `json:"sort_order"`
}

// UpdateEducationRequest is the request payload to update an education.
type UpdateEducationRequest struct {
	Institution  *string `json:"institution,omitempty" binding:"omitempty,max=255"`
	Degree       *string `json:"degree,omitempty" binding:"omitempty,max=255"`
	FieldOfStudy *string `json:"field_of_study,omitempty" binding:"omitempty,max=255"`
	StartDate    *string `json:"start_date,omitempty" binding:"omitempty,max=20"`
	EndDate      *string `json:"end_date,omitempty" binding:"omitempty,max=20"`
	IsCurrent    *bool   `json:"is_current,omitempty"`
	GPA          *string `json:"gpa,omitempty" binding:"omitempty,max=20"`
	Description  *string `json:"description,omitempty" binding:"omitempty,max=10000"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

// CreateSkillRequest is the request payload to add a skill.
type CreateSkillRequest struct {
	Name      string `json:"name" binding:"max=100"`
	Level     string `json:"level" binding:"max=50"`
	SortOrder int    `json:"sort_order"`
}

// UpdateSkillRequest is the request payload to update a skill.
type UpdateSkillRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,max=100"`
	Level     *string `json:"level,omitempty" binding:"omitempty,max=50"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// CreateLanguageRequest is the request payload to add a language.
type CreateLanguageRequest struct {
	Name        string `json:"name" binding:"max=100"`
	Proficiency string `json:"proficiency" binding:"max=50"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateLanguageRequest is the request payload to update a language.
type UpdateLanguageRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,max=100"`
	Proficiency *string `json:"proficiency,omitempty" binding:"omitempty,max=50"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// CreateCertificationRequest is the request payload to add a certification.
type CreateCertificationRequest struct {
	Name       string `json:"name" binding:"max=255"`
	Issuer     string `json:"issuer" binding:"max=255"`
	IssueDate  string `json:"issue_date" binding:"max=20"`
	ExpiryDate string `json:"expiry_date" binding:"max=20"`
	URL        string `json:"url" binding:"max=500"`
	SortOrder  int    `json:"sort_order"`
}

// UpdateCertificationRequest is the request payload to update a certification.
type UpdateCertificationRequest struct {
	Name       *string `json:"name,omitempty" binding:"omitempty,max=255"`
	Issuer     *string `json:"issuer,omitempty" binding:"omitempty,max=255"`
	IssueDate  *string `json:"issue_date,omitempty" binding:"omitempty,max=20"`
	ExpiryDate *string `json:"expiry_date,omitempty" binding:"omitempty,max=20"`
	URL        *string `json:"url,omitempty" binding:"omitempty,max=500"`
	SortOrder  *int    `json:"sort_order,omitempty"`
}

// CreateProjectRequest is the request payload to add a project.
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"max=255"`
	URL         string `json:"url" binding:"max=500"`
	StartDate   string `json:"start_date" binding:"max=20"`
	EndDate     string `json:"end_date" binding:"max=20"`
	Description string `json:"description" binding:"max=10000"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateProjectRequest is the request payload to update a project.
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,max=255"`
	URL         *string `json:"url,omitempty" binding:"omitempty,max=500"`
	StartDate   *string `json:"start_date,omitempty" binding:"omitempty,max=20"`
	EndDate     *string `json:"end_date,omitempty" binding:"omitempty,max=20"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=10000"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// CreateVolunteeringRequest is the request payload to add a volunteering entry.
type CreateVolunteeringRequest struct {
	Organization string `json:"organization" binding:"max=255"`
	Role         string `json:"role" binding:"max=255"`
	StartDate    string `json:"start_date" binding:"max=20"`
	EndDate      string `json:"end_date" binding:"max=20"`
	Description  string `json:"description" binding:"max=10000"`
	SortOrder    int    `json:"sort_order"`
}

// UpdateVolunteeringRequest is the request payload to update a volunteering entry.
type UpdateVolunteeringRequest struct {
	Organization *string `json:"organization,omitempty" binding:"omitempty,max=255"`
	Role         *string `json:"role,omitempty" binding:"omitempty,max=255"`
	StartDate    *string `json:"start_date,omitempty" binding:"omitempty,max=20"`
	EndDate      *string `json:"end_date,omitempty" binding:"omitempty,max=20"`
	Description  *string `json:"description,omitempty" binding:"omitempty,max=10000"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

// CreateCustomSectionRequest is the request payload to add a custom section.
type CreateCustomSectionRequest struct {
	Title     string `json:"title" binding:"max=255"`
	Content   string `json:"content" binding:"max=20000"`
	SortOrder int    `json:"sort_order"`
}

// UpdateCustomSectionRequest is the request payload to update a custom section.
type UpdateCustomSectionRequest struct {
	Title     *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Content   *string `json:"content,omitempty" binding:"omitempty,max=20000"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// UpdateSectionOrderRequest is a single section order entry.
type UpdateSectionOrderRequest struct {
	SectionKey string `json:"section_key" binding:"required,max=50"`
	SortOrder  int    `json:"sort_order"`
	IsVisible  bool   `json:"is_visible"`
	Column     string `json:"column"`
}

// BatchUpdateSectionOrderRequest is the request payload to batch update section ordering.
type BatchUpdateSectionOrderRequest struct {
	Sections []UpdateSectionOrderRequest `json:"sections" binding:"required,max=20"`
}
