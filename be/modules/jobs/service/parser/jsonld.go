package parser

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"
)

// jobPostingLD represents a schema.org/JobPosting in JSON-LD format
type jobPostingLD struct {
	Context     interface{} `json:"@context"`
	Type        string      `json:"@type"`
	Title       string      `json:"title"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	HiringOrg   interface{} `json:"hiringOrganization"`
	JobLocation interface{} `json:"jobLocation"`
}

// hiringOrganization can be a string or an object
type hiringOrganization struct {
	Type string `json:"@type"`
	Name string `json:"name"`
}

// ExtractJobPostingLD extracts JSON-LD JobPosting data from HTML
func ExtractJobPostingLD(htmlContent []byte) (*jobPostingLD, bool) {
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, false
	}

	scripts := findJSONLDScripts(doc)
	for _, script := range scripts {
		posting := tryParseJobPosting(script)
		if posting != nil {
			return posting, true
		}
	}

	return nil, false
}

// findJSONLDScripts extracts all <script type="application/ld+json"> content
func findJSONLDScripts(n *html.Node) []string {
	var scripts []string

	if n.Type == html.ElementNode && n.Data == "script" {
		for _, attr := range n.Attr {
			if attr.Key == "type" && attr.Val == "application/ld+json" {
				if n.FirstChild != nil {
					scripts = append(scripts, n.FirstChild.Data)
				}
				break
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		scripts = append(scripts, findJSONLDScripts(c)...)
	}

	return scripts
}

// tryParseJobPosting tries to parse a JSON string as a JobPosting
func tryParseJobPosting(jsonStr string) *jobPostingLD {
	// Try single object first
	var single jobPostingLD
	if err := json.Unmarshal([]byte(jsonStr), &single); err == nil {
		if isJobPosting(&single) {
			return &single
		}
	}

	// Try array of objects
	var arr []jobPostingLD
	if err := json.Unmarshal([]byte(jsonStr), &arr); err == nil {
		for i := range arr {
			if isJobPosting(&arr[i]) {
				return &arr[i]
			}
		}
	}

	// Try @graph pattern
	var graph struct {
		Graph []jobPostingLD `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &graph); err == nil {
		for i := range graph.Graph {
			if isJobPosting(&graph.Graph[i]) {
				return &graph.Graph[i]
			}
		}
	}

	return nil
}

func isJobPosting(ld *jobPostingLD) bool {
	return ld.Type == "JobPosting"
}

// GetTitle returns the job title from JSON-LD (prefers "title", falls back to "name")
func (ld *jobPostingLD) GetTitle() string {
	if ld.Title != "" {
		return ld.Title
	}
	return ld.Name
}

// GetCompanyName extracts the company name from hiringOrganization
func (ld *jobPostingLD) GetCompanyName() string {
	if ld.HiringOrg == nil {
		return ""
	}

	switch v := ld.HiringOrg.(type) {
	case string:
		return v
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok {
			return name
		}
	}
	return ""
}

// GetLocation extracts the location from jobLocation
func (ld *jobPostingLD) GetLocation() string {
	if ld.JobLocation == nil {
		return ""
	}

	switch v := ld.JobLocation.(type) {
	case string:
		return v
	case map[string]interface{}:
		return extractAddress(v)
	case []interface{}:
		if len(v) > 0 {
			if loc, ok := v[0].(map[string]interface{}); ok {
				return extractAddress(loc)
			}
		}
	}
	return ""
}

func extractAddress(loc map[string]interface{}) string {
	// Try address sub-object first
	if addr, ok := loc["address"].(map[string]interface{}); ok {
		parts := []string{}
		if city, ok := addr["addressLocality"].(string); ok && city != "" {
			parts = append(parts, city)
		}
		if region, ok := addr["addressRegion"].(string); ok && region != "" {
			parts = append(parts, region)
		}
		if country, ok := addr["addressCountry"].(string); ok && country != "" {
			parts = append(parts, country)
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
	}

	// Try name field
	if name, ok := loc["name"].(string); ok {
		return name
	}

	return ""
}
