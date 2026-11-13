package service

import "time"

// Frozen storage schema. Keep v1 unchanged; add a version and migration for schema changes.
type snapshotMetadataV1 struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	TemplateID   string    `json:"template_id"`
	FontFamily   string    `json:"font_family"`
	PrimaryColor string    `json:"primary_color"`
	TextColor    string    `json:"text_color"`
	Spacing      int       `json:"spacing"`
	MarginTop    int       `json:"margin_top"`
	MarginBottom int       `json:"margin_bottom"`
	MarginLeft   int       `json:"margin_left"`
	MarginRight  int       `json:"margin_right"`
	LayoutMode   string    `json:"layout_mode"`
	SidebarWidth int       `json:"sidebar_width"`
	FontSize     int       `json:"font_size"`
	SkillDisplay string    `json:"skill_display"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type snapshotContactV1 struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Website  string `json:"website"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
}

type snapshotSummaryV1 struct {
	Content string `json:"content"`
}

type snapshotExperienceV1 struct {
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

type snapshotEducationV1 struct {
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

type snapshotSkillV1 struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Level     string `json:"level"`
	SortOrder int    `json:"sort_order"`
}

type snapshotLanguageV1 struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
	SortOrder   int    `json:"sort_order"`
}

type snapshotCertificationV1 struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Issuer     string `json:"issuer"`
	IssueDate  string `json:"issue_date"`
	ExpiryDate string `json:"expiry_date"`
	URL        string `json:"url"`
	SortOrder  int    `json:"sort_order"`
}

type snapshotProjectV1 struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type snapshotVolunteeringV1 struct {
	ID           string `json:"id"`
	Organization string `json:"organization"`
	Role         string `json:"role"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Description  string `json:"description"`
	SortOrder    int    `json:"sort_order"`
}

type snapshotCustomSectionV1 struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
}

type snapshotSectionOrderV1 struct {
	SectionKey string `json:"section_key"`
	SortOrder  int    `json:"sort_order"`
	IsVisible  bool   `json:"is_visible"`
	Column     string `json:"column"`
}
