package pdf

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// formatDate formats a date string for display.
// Input: "2024-01" or "2024-01-15" -> "Jan 2024"
func formatDate(date string) string {
	if date == "" {
		return ""
	}
	parts := strings.Split(date, "-")
	if len(parts) < 2 {
		return date
	}
	months := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return date
	}
	return fmt.Sprintf("%s %s", months[month], parts[0])
}

// dateRange creates a date range string.
func dateRange(start, end string, isCurrent bool) string {
	s := formatDate(start)
	if s == "" {
		return ""
	}
	if isCurrent {
		return s + " — Present"
	}
	e := formatDate(end)
	if e == "" {
		return s
	}
	return s + " — " + e
}

// safeHTML marks a string as safe HTML.
func safeHTML(s string) template.HTML {
	// Convert newlines to <br> for whitespace preservation
	escaped := template.HTMLEscapeString(s)
	return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>"))
}

// joinSkills joins skills into a comma-separated string.
func joinSkills(skills []*model.SkillDTO) string {
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, " · ")
}

// joinLanguages joins languages with proficiency into a string.
func joinLanguages(languages []*model.LanguageDTO) string {
	items := make([]string, 0, len(languages))
	for _, l := range languages {
		if l.Name == "" {
			continue
		}
		if l.Proficiency != "" {
			items = append(items, fmt.Sprintf("%s (%s)", l.Name, l.Proficiency))
		} else {
			items = append(items, l.Name)
		}
	}
	return strings.Join(items, " · ")
}

// safeURL returns the URL as template.URL if it has a safe scheme (http/https),
// or empty string otherwise. Returning template.URL tells html/template the
// value has been sanitised and prevents ZgotmplZ guards in href attributes.
func safeURL(s string) template.URL {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		return template.URL(s)
	}
	return ""
}

// lightenColor lightens a hex color for background use.
func lightenColor(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return "#f0f4f8"
	}
	r, _ := strconv.ParseInt(hex[1:3], 16, 64)
	g, _ := strconv.ParseInt(hex[3:5], 16, 64)
	b, _ := strconv.ParseInt(hex[5:7], 16, 64)

	// Mix with white (factor 0.85)
	r = r + (255-r)*85/100
	g = g + (255-g)*85/100
	b = b + (255-b)*85/100

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// contrastColor returns white or dark text for best contrast.
func contrastColor(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return "#ffffff"
	}
	r, _ := strconv.ParseInt(hex[1:3], 16, 64)
	g, _ := strconv.ParseInt(hex[3:5], 16, 64)
	b, _ := strconv.ParseInt(hex[5:7], 16, 64)

	// Relative luminance
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
	if luminance > 0.5 {
		return "#1a1a1a"
	}
	return "#ffffff"
}
