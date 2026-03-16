package docx

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/gomutex/godocx"
	"github.com/gomutex/godocx/docx"
)

// DOCXService generates DOCX documents from resume data.
type DOCXService struct{}

// NewDOCXService creates a new DOCXService.
func NewDOCXService() *DOCXService {
	return &DOCXService{}
}

// GenerateResumeDOCX generates an ATS-friendly DOCX document from the given resume data.
func (s *DOCXService) GenerateResumeDOCX(data *model.FullResumeDTO) ([]byte, error) {
	doc, err := godocx.NewDocument()
	if err != nil {
		return nil, fmt.Errorf("failed to create docx document: %w", err)
	}

	// Add contact name as heading
	addContactHeading(doc, data.Contact)

	// Add contact info line
	addContactInfo(doc, data.Contact)

	// Build visible sections ordered by section_order
	visibleSections := buildVisibleSections(data.SectionOrder)

	// Render each visible section in order
	for _, sectionKey := range visibleSections {
		addSection(doc, sectionKey, data)
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write docx: %w", err)
	}

	return buf.Bytes(), nil
}

// CoverLetterDOCXData holds the data needed to generate a cover letter DOCX.
type CoverLetterDOCXData struct {
	RecipientName  string
	RecipientTitle string
	CompanyName    string
	CompanyAddress string
	Date           string
	Greeting       string
	Paragraphs     []string
	Closing        string
}

// GenerateCoverLetterDOCX generates an ATS-friendly DOCX document from cover letter data.
func (s *DOCXService) GenerateCoverLetterDOCX(data *CoverLetterDOCXData) ([]byte, error) {
	doc, err := godocx.NewDocument()
	if err != nil {
		return nil, fmt.Errorf("failed to create docx document: %w", err)
	}

	// Add recipient info as heading
	recipientLine := buildRecipientLine(data.RecipientName, data.RecipientTitle, data.CompanyName)
	if recipientLine != "" {
		_, _ = doc.AddHeading(recipientLine, 0)
	}

	// Add company address
	if data.CompanyAddress != "" {
		doc.AddParagraph(data.CompanyAddress)
	}

	// Add date
	if data.Date != "" {
		doc.AddParagraph(data.Date)
	}

	// Add greeting
	if data.Greeting != "" {
		p := doc.AddEmptyParagraph()
		p.AddText(data.Greeting).Bold(true)
	}

	// Add paragraphs
	for _, paragraph := range data.Paragraphs {
		if paragraph != "" {
			doc.AddParagraph(paragraph)
		}
	}

	// Add closing
	if data.Closing != "" {
		p := doc.AddEmptyParagraph()
		p.AddText(data.Closing).Bold(true)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write docx: %w", err)
	}

	return buf.Bytes(), nil
}

// buildRecipientLine constructs a recipient heading line.
func buildRecipientLine(name, title, company string) string {
	parts := make([]string, 0, 3)
	if name != "" {
		parts = append(parts, name)
	}
	if title != "" {
		parts = append(parts, title)
	}
	if company != "" {
		parts = append(parts, company)
	}
	return strings.Join(parts, ", ")
}

// buildVisibleSections returns section keys sorted by sort_order, filtered to only visible sections.
func buildVisibleSections(sectionOrder []*model.SectionOrderDTO) []string {
	type orderedSection struct {
		key       string
		sortOrder int
	}

	var visible []orderedSection
	for _, so := range sectionOrder {
		if so.IsVisible {
			visible = append(visible, orderedSection{
				key:       so.SectionKey,
				sortOrder: so.SortOrder,
			})
		}
	}

	sort.Slice(visible, func(i, j int) bool {
		return visible[i].sortOrder < visible[j].sortOrder
	})

	keys := make([]string, len(visible))
	for i, v := range visible {
		keys[i] = v.key
	}
	return keys
}

// addContactHeading adds the contact full name as the document title heading.
func addContactHeading(doc *docx.RootDoc, contact *model.ContactDTO) {
	if contact == nil || contact.FullName == "" {
		return
	}
	// Level 0 = Title style
	_, _ = doc.AddHeading(contact.FullName, 0)
}

// addContactInfo adds a single line with email, phone, location, and linkedin.
func addContactInfo(doc *docx.RootDoc, contact *model.ContactDTO) {
	if contact == nil {
		return
	}

	parts := make([]string, 0, 6)
	if contact.Email != "" {
		parts = append(parts, contact.Email)
	}
	if contact.Phone != "" {
		parts = append(parts, contact.Phone)
	}
	if contact.Location != "" {
		parts = append(parts, contact.Location)
	}
	if contact.LinkedIn != "" {
		parts = append(parts, contact.LinkedIn)
	}
	if contact.Website != "" {
		parts = append(parts, contact.Website)
	}
	if contact.GitHub != "" {
		parts = append(parts, contact.GitHub)
	}

	if len(parts) == 0 {
		return
	}

	doc.AddParagraph(strings.Join(parts, " | "))
}

// addSection dispatches to the appropriate section renderer based on the section key.
func addSection(doc *docx.RootDoc, sectionKey string, data *model.FullResumeDTO) {
	switch sectionKey {
	case "summary":
		addSummarySection(doc, data.Summary)
	case "experience":
		addExperienceSection(doc, data.Experiences)
	case "education":
		addEducationSection(doc, data.Educations)
	case "skills":
		addSkillsSection(doc, data.Skills)
	case "languages":
		addLanguagesSection(doc, data.Languages)
	case "certifications":
		addCertificationsSection(doc, data.Certifications)
	case "projects":
		addProjectsSection(doc, data.Projects)
	case "volunteering":
		addVolunteeringSection(doc, data.Volunteering)
	case "custom_sections":
		addCustomSectionsSection(doc, data.CustomSections)
	}
}

// addSectionHeading adds a bold, uppercase heading for a section.
func addSectionHeading(doc *docx.RootDoc, title string) {
	heading, err := doc.AddHeading(strings.ToUpper(title), 1)
	if err != nil {
		// Fallback to a bold paragraph if heading creation fails
		p := doc.AddParagraph("")
		p.AddText(strings.ToUpper(title)).Bold(true).Size(14)
		return
	}
	_ = heading
}

// addSummarySection adds the summary section.
func addSummarySection(doc *docx.RootDoc, summary *model.SummaryDTO) {
	if summary == nil || summary.Content == "" {
		return
	}
	addSectionHeading(doc, "Summary")
	doc.AddParagraph(summary.Content)
}

// addExperienceSection adds all experience entries.
func addExperienceSection(doc *docx.RootDoc, experiences []*model.ExperienceDTO) {
	if len(experiences) == 0 {
		return
	}
	addSectionHeading(doc, "Experience")

	for _, exp := range experiences {
		// Title line: Position at Company (Start - End)
		titleLine := buildTitleLine(exp.Position, exp.Company, exp.StartDate, exp.EndDate, exp.IsCurrent)
		p := doc.AddEmptyParagraph()
		p.AddText(titleLine).Bold(true)

		// Location
		if exp.Location != "" {
			doc.AddParagraph(exp.Location)
		}

		// Description
		if exp.Description != "" {
			doc.AddParagraph(exp.Description)
		}
	}
}

// addEducationSection adds all education entries.
func addEducationSection(doc *docx.RootDoc, educations []*model.EducationDTO) {
	if len(educations) == 0 {
		return
	}
	addSectionHeading(doc, "Education")

	for _, edu := range educations {
		// Title line: Degree in Field at Institution (Start - End)
		titleLine := buildEducationTitleLine(edu)
		p := doc.AddEmptyParagraph()
		p.AddText(titleLine).Bold(true)

		// GPA
		if edu.GPA != "" {
			doc.AddParagraph("GPA: " + edu.GPA)
		}

		// Description
		if edu.Description != "" {
			doc.AddParagraph(edu.Description)
		}
	}
}

// addSkillsSection adds skills as a comma-separated list with levels.
func addSkillsSection(doc *docx.RootDoc, skills []*model.SkillDTO) {
	if len(skills) == 0 {
		return
	}
	addSectionHeading(doc, "Skills")

	parts := make([]string, 0, len(skills))
	for _, skill := range skills {
		entry := skill.Name
		if skill.Level != "" {
			entry += " (" + skill.Level + ")"
		}
		parts = append(parts, entry)
	}
	doc.AddParagraph(strings.Join(parts, ", "))
}

// addLanguagesSection adds languages as a comma-separated list with proficiency.
func addLanguagesSection(doc *docx.RootDoc, languages []*model.LanguageDTO) {
	if len(languages) == 0 {
		return
	}
	addSectionHeading(doc, "Languages")

	parts := make([]string, 0, len(languages))
	for _, lang := range languages {
		entry := lang.Name
		if lang.Proficiency != "" {
			entry += " (" + lang.Proficiency + ")"
		}
		parts = append(parts, entry)
	}
	doc.AddParagraph(strings.Join(parts, ", "))
}

// addCertificationsSection adds all certification entries.
func addCertificationsSection(doc *docx.RootDoc, certifications []*model.CertificationDTO) {
	if len(certifications) == 0 {
		return
	}
	addSectionHeading(doc, "Certifications")

	for _, cert := range certifications {
		// Name - Issuer (Date)
		line := cert.Name
		if cert.Issuer != "" {
			line += " - " + cert.Issuer
		}
		if cert.IssueDate != "" {
			line += " (" + cert.IssueDate + ")"
		}
		doc.AddParagraph(line)
	}
}

// addProjectsSection adds all project entries.
func addProjectsSection(doc *docx.RootDoc, projects []*model.ProjectDTO) {
	if len(projects) == 0 {
		return
	}
	addSectionHeading(doc, "Projects")

	for _, proj := range projects {
		// Name (URL)
		titleLine := proj.Name
		if proj.URL != "" {
			titleLine += " (" + proj.URL + ")"
		}
		p := doc.AddEmptyParagraph()
		p.AddText(titleLine).Bold(true)

		// Description
		if proj.Description != "" {
			doc.AddParagraph(proj.Description)
		}
	}
}

// addVolunteeringSection adds all volunteering entries.
func addVolunteeringSection(doc *docx.RootDoc, volunteering []*model.VolunteeringDTO) {
	if len(volunteering) == 0 {
		return
	}
	addSectionHeading(doc, "Volunteering")

	for _, vol := range volunteering {
		// Role at Organization (Start - End)
		titleLine := buildTitleLine(vol.Role, vol.Organization, vol.StartDate, vol.EndDate, false)
		p := doc.AddEmptyParagraph()
		p.AddText(titleLine).Bold(true)

		// Description
		if vol.Description != "" {
			doc.AddParagraph(vol.Description)
		}
	}
}

// addCustomSectionsSection adds all custom section entries.
func addCustomSectionsSection(doc *docx.RootDoc, customSections []*model.CustomSectionDTO) {
	if len(customSections) == 0 {
		return
	}

	for _, cs := range customSections {
		if cs.Title != "" {
			addSectionHeading(doc, cs.Title)
		}
		if cs.Content != "" {
			doc.AddParagraph(cs.Content)
		}
	}
}

// buildTitleLine constructs a formatted title line like "Position at Company (Start - End)".
func buildTitleLine(role, organization, startDate, endDate string, isCurrent bool) string {
	var sb strings.Builder

	if role != "" {
		sb.WriteString(role)
	}
	if organization != "" {
		if sb.Len() > 0 {
			sb.WriteString(" at ")
		}
		sb.WriteString(organization)
	}

	dateRange := formatDateRange(startDate, endDate, isCurrent)
	if dateRange != "" {
		sb.WriteString(" (")
		sb.WriteString(dateRange)
		sb.WriteString(")")
	}

	return sb.String()
}

// buildEducationTitleLine constructs a formatted education title line.
func buildEducationTitleLine(edu *model.EducationDTO) string {
	var sb strings.Builder

	if edu.Degree != "" {
		sb.WriteString(edu.Degree)
	}
	if edu.FieldOfStudy != "" {
		if sb.Len() > 0 {
			sb.WriteString(" in ")
		}
		sb.WriteString(edu.FieldOfStudy)
	}
	if edu.Institution != "" {
		if sb.Len() > 0 {
			sb.WriteString(" at ")
		}
		sb.WriteString(edu.Institution)
	}

	dateRange := formatDateRange(edu.StartDate, edu.EndDate, edu.IsCurrent)
	if dateRange != "" {
		sb.WriteString(" (")
		sb.WriteString(dateRange)
		sb.WriteString(")")
	}

	return sb.String()
}

// formatDateRange formats a date range string from start and end dates.
func formatDateRange(startDate, endDate string, isCurrent bool) string {
	if startDate == "" && endDate == "" && !isCurrent {
		return ""
	}

	start := startDate
	if start == "" {
		start = ""
	}

	end := endDate
	if isCurrent {
		end = "Present"
	}

	if start != "" && end != "" {
		return start + " - " + end
	}
	if start != "" {
		return start
	}
	return end
}
