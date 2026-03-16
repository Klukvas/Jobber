package model

import "time"

// ResumeBuilder represents the master resume builder record.
type ResumeBuilder struct {
	ID           string
	UserID       string
	Title        string
	TemplateID   string
	FontFamily   string
	PrimaryColor string
	TextColor    string
	Spacing      int
	MarginTop    int
	MarginBottom int
	MarginLeft   int
	MarginRight  int
	LayoutMode   string
	SidebarWidth int
	FontSize     int
	SkillDisplay string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ResumeBuilderDTO is the JSON response for a resume builder.
type ResumeBuilderDTO struct {
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

// ToDTO converts a ResumeBuilder to ResumeBuilderDTO.
func (r *ResumeBuilder) ToDTO() *ResumeBuilderDTO {
	return &ResumeBuilderDTO{
		ID:           r.ID,
		Title:        r.Title,
		TemplateID:   r.TemplateID,
		FontFamily:   r.FontFamily,
		PrimaryColor: r.PrimaryColor,
		TextColor:    r.TextColor,
		Spacing:      r.Spacing,
		MarginTop:    r.MarginTop,
		MarginBottom: r.MarginBottom,
		MarginLeft:   r.MarginLeft,
		MarginRight:  r.MarginRight,
		LayoutMode:   r.LayoutMode,
		SidebarWidth: r.SidebarWidth,
		FontSize:     r.FontSize,
		SkillDisplay: r.SkillDisplay,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// ResumeTemplate represents a template definition.
type ResumeTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsPremium   bool   `json:"is_premium"`
}
