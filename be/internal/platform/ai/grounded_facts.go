package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Grounding units retain complete paragraphs and their actor/context. Splitting at
// numbers or isolated keywords would let the same metric describe a different fact.
type groundedFact struct {
	ID                string `json:"id"`
	Text              string `json:"text"`
	Section           string `json:"section"`
	EntryID           string `json:"-"`
	Actor             string `json:"actor,omitempty"`
	SourceText        string `json:"-"`
	ExperienceRole    string `json:"-"`
	ExperienceCompany string `json:"-"`
}
type groundingSource struct {
	Summary *struct {
		Content string `json:"content"`
	} `json:"summary"`
	Experiences []struct {
		ID, Company, Position, Description string
		StartDate                          string `json:"start_date"`
		EndDate                            string `json:"end_date"`
	} `json:"experiences"`
	Skills     []struct{ ID, Name, Level string } `json:"skills"`
	Educations []struct {
		Institution, Degree, FieldOfStudy string
		Field                             string `json:"field_of_study"`
		StartDate                         string `json:"start_date"`
		EndDate                           string `json:"end_date"`
		IsCurrent                         *bool  `json:"is_current"`
		GPA, Description                  string
	} `json:"educations"`
	Languages      []struct{ Name, Proficiency string } `json:"languages"`
	Certifications []struct {
		Name, Issuer string
		IssueDate    string `json:"issue_date"`
		ExpiryDate   string `json:"expiry_date"`
		URL          string
	} `json:"certifications"`
	Projects []struct {
		Name, Description, URL string
		StartDate              string `json:"start_date"`
		EndDate                string `json:"end_date"`
	} `json:"projects"`
	Volunteering []struct {
		Organization, Role, Description string
		StartDate                       string `json:"start_date"`
		EndDate                         string `json:"end_date"`
	} `json:"volunteering"`
	CustomSections []struct{ Title, Content string } `json:"custom_sections"`
}
type groundingCatalog struct {
	Facts       []groundedFact    `json:"facts"`
	Experiences map[string]string `json:"experience_ids,omitempty"`
	Skills      []groundedFact    `json:"skills,omitempty"`
}

func paragraphFacts(text, section, entry, actor, prefix string) []groundedFact {
	text = strings.TrimSpace(text)
	if text == "" {
		return []groundedFact{}
	}
	return []groundedFact{{ID: prefix + "0", Text: text, Section: section, EntryID: entry, Actor: actor}}
}
func resumeGroundingCatalog(text string, strict bool) (groundingCatalog, error) {
	c := groundingCatalog{Facts: []groundedFact{}, Experiences: map[string]string{}, Skills: []groundedFact{}}
	var source groundingSource
	if err := json.Unmarshal([]byte(text), &source); err != nil {
		if strict {
			return c, fmt.Errorf("invalid structured resume facts: %w", err)
		}
		c.Facts = paragraphFacts(text, "resume", "", "", "f")
		return c, nil
	}
	if source.Summary != nil {
		c.Facts = append(c.Facts, paragraphFacts(source.Summary.Content, "summary", "", "", "s")...)
	}
	for i, e := range source.Experiences {
		id := fmt.Sprintf("e%d", i)
		if e.ID == "" {
			e.ID = id
		}
		c.Experiences[id] = e.ID
		if strings.TrimSpace(e.Description) != "" {
			c.Facts = append(c.Facts, groundedFact{ID: id + "f0", Text: GroundedExperienceContext(e.Position, e.Company, e.StartDate, e.EndDate, e.Description), SourceText: e.Description, ExperienceRole: e.Position, ExperienceCompany: e.Company, Section: "experience", EntryID: e.ID})
		}
	}
	for i, s := range source.Skills {
		id := fmt.Sprintf("k%d", i)
		if s.ID == "" {
			s.ID = id
		}
		credential := s.Name
		if s.Level != "" {
			credential += " (listed level: " + s.Level + ")"
		}
		c.Skills = append(c.Skills, groundedFact{ID: id, Text: s.Name, SourceText: credential, Section: "skills", EntryID: s.ID})
	}
	if strict {
		return c, nil
	}
	for _, e := range source.Educations {
		parts := []string{}
		for _, field := range []struct{ label, value string }{{"Degree", e.Degree}, {"Field of study", e.Field}, {"Field of study", e.FieldOfStudy}, {"Institution", e.Institution}} {
			if strings.TrimSpace(field.value) != "" {
				parts = append(parts, field.label+": "+field.value)
			}
		}
		if strings.TrimSpace(e.StartDate) != "" || strings.TrimSpace(e.EndDate) != "" {
			parts = append(parts, "Dates: "+e.StartDate+" - "+e.EndDate)
		}
		if e.IsCurrent != nil {
			parts = append(parts, fmt.Sprintf("Current study: %t", *e.IsCurrent))
		}
		for _, field := range []struct{ label, value string }{{"GPA", e.GPA}, {"Description", e.Description}} {
			if strings.TrimSpace(field.value) != "" {
				parts = append(parts, field.label+": "+field.value)
			}
		}
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Degree + " " + e.Field + " " + e.FieldOfStudy + " at " + e.Institution), SourceText: strings.Join(parts, "; "), Section: "education"})
	}
	for _, e := range source.Languages {
		full := completeSourceFields([2]string{"Language", e.Name}, [2]string{"Listed proficiency", e.Proficiency})
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Name + " " + e.Proficiency), SourceText: full, Section: "languages"})
	}
	for _, e := range source.Certifications {
		full := completeSourceFields([2]string{"Certification", e.Name}, [2]string{"Issuer", e.Issuer}, [2]string{"Issue date", e.IssueDate}, [2]string{"Expiry date", e.ExpiryDate}, [2]string{"URL", e.URL})
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Name + " — " + e.Issuer + " " + e.IssueDate), SourceText: full, Section: "certifications"})
	}
	for _, e := range source.Projects {
		full := completeSourceFields([2]string{"Project", e.Name}, [2]string{"URL", e.URL}, [2]string{"Start date", e.StartDate}, [2]string{"End date", e.EndDate}, [2]string{"Description", e.Description})
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Description), SourceText: full, Section: "projects", Actor: e.Name})
	}
	for _, e := range source.Volunteering {
		full := completeSourceFields([2]string{"Organization", e.Organization}, [2]string{"Role", e.Role}, [2]string{"Start date", e.StartDate}, [2]string{"End date", e.EndDate}, [2]string{"Description", e.Description})
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Description), SourceText: full, Section: "volunteering", Actor: strings.TrimSpace(e.Role + " at " + e.Organization)})
	}
	for _, e := range source.CustomSections {
		full := completeSourceFields([2]string{"Section", e.Title}, [2]string{"Content", e.Content})
		c.Facts = append(c.Facts, groundedFact{Text: strings.TrimSpace(e.Content), SourceText: full, Section: "custom", Actor: e.Title})
	}
	c.Facts = append(c.Facts, c.Skills...)
	facts := make([]groundedFact, 0, len(c.Facts))
	for _, fact := range c.Facts {
		if fact.Section != "experience" && fact.Section != "summary" && fact.Section != "skills" {
			// Optional rows may have metadata without a description. Preserve
			// that complete context; entirely blank rows supply no evidence.
			if strings.TrimSpace(fact.SourceText) == "" {
				continue
			}
			if strings.TrimSpace(fact.Text) == "" {
				fact.Text = fact.SourceText
			}
		}
		fact.ID = fmt.Sprintf("f%d", len(facts))
		facts = append(facts, fact)
	}
	c.Facts = facts
	// Keep the measured experienced-profile catalog unchanged. Credentials are
	// expanded only for profiles with no complete work description to select.
	if !hasLetterExperience(c) {
		for i, fact := range c.Facts {
			switch fact.Section {
			case "summary":
				c.Facts[i].SourceText = fact.Text
			case "skills", "education", "languages", "certifications", "projects", "volunteering", "custom":
				c.Facts[i].Text = fact.SourceText
			}
		}
	}
	return c, nil
}

// Labels keep metadata attached to the same complete source entry. Missing fields
// stay absent; field values, including restrictions and line breaks, stay verbatim.
func completeSourceFields(fields ...[2]string) string {
	parts := []string{}
	for _, field := range fields {
		if strings.TrimSpace(field[1]) != "" {
			parts = append(parts, field[0]+": "+field[1])
		}
	}
	return strings.Join(parts, "; ")
}
func resolveFacts(c groundingCatalog, ids []string, section, entry string) ([]groundedFact, error) {
	if len(ids) == 0 || len(ids) > 6 {
		return nil, fmt.Errorf("invalid fact selection")
	}
	byID := map[string]groundedFact{}
	for _, f := range c.Facts {
		byID[f.ID] = f
	}
	selected := make([]groundedFact, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		f, ok := byID[id]
		if !ok || seen[id] || (section != "" && (f.Section != section || f.EntryID != entry)) {
			return nil, fmt.Errorf("unknown, repeated or mismatched fact")
		}
		seen[id] = true
		selected = append(selected, f)
	}
	return selected, nil
}
func groundedText(facts []groundedFact) string {
	parts := make([]string, 0, len(facts))
	for _, f := range facts {
		parts = append(parts, f.Text)
	}
	return strings.Join(parts, "\n")
}
func validateGroundingInput(c groundingCatalog, maxBytes int) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if len(data) > maxBytes || !utf8.Valid(data) {
		return fmt.Errorf("grounding source exceeds request limit")
	}
	for _, fact := range c.Facts {
		if strings.ContainsRune(fact.SourceText, '\x00') || len([]rune(fact.SourceText)) > 4000 {
			return fmt.Errorf("invalid or oversized complete source entry")
		}
	}
	for _, f := range append(append([]groundedFact{}, c.Facts...), c.Skills...) {
		if strings.TrimSpace(f.Text) == "" || strings.ContainsRune(f.Text, '\x00') || len([]rune(f.Text)) > 4000 {
			return fmt.Errorf("invalid or oversized source paragraph")
		}
	}
	return nil
}

// GroundedExperienceContext preserves the entire description, including qualifiers
// across line breaks. Metadata is copied verbatim from the same source entry.
func GroundedExperienceContext(position, company, start, end, description string) string {
	fields := []string{}
	if position != "" {
		fields = append(fields, position)
	}
	if company != "" {
		fields = append(fields, "at "+company)
	}
	if start != "" || end != "" {
		fields = append(fields, "["+start+" - "+end+"]")
	}
	if len(fields) == 0 {
		return description
	}
	return strings.Join(fields, " ") + ":\n" + description
}
