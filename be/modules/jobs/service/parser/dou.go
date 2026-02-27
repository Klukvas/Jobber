package parser

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// DOUParser parses job postings from DOU.ua
type DOUParser struct {
	fetcher *Fetcher
}

// NewDOUParser creates a new DOU parser
func NewDOUParser(fetcher *Fetcher) *DOUParser {
	return &DOUParser{fetcher: fetcher}
}

// CanParse checks if the URL is a DOU job posting
func (p *DOUParser) CanParse(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "dou.ua" || host == "jobs.dou.ua" ||
		strings.HasSuffix(host, ".dou.ua")
}

// Parse extracts job data from a DOU URL
func (p *DOUParser) Parse(ctx context.Context, url string) (*ParsedJob, error) {
	body, err := p.fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	job := &ParsedJob{
		Source: "dou",
		URL:    url,
	}

	// DOU uses standard HTML. Try extracting from known elements.
	// Title is typically in <h1 class="g-h2">
	job.Title = findElementText(doc, "h1", "g-h2")
	if job.Title == "" {
		// Fallback to og:title
		job.Title = findMeta(doc, "og:title")
	}

	// Company is in <a class="company"> or from the breadcrumb
	job.CompanyName = findElementText(doc, "a", "company")

	// Location from <span class="place">
	job.Location = findElementText(doc, "span", "place")
	if job.Location == "" {
		job.Location = findElementText(doc, "div", "place")
	}

	// Description from <div class="b-typo vacancy-section">
	job.Description = findMeta(doc, "og:description")

	// Clean up DOU title: "Title | DOU"
	if idx := strings.LastIndex(job.Title, " | DOU"); idx > 0 {
		job.Title = job.Title[:idx]
	}
	if idx := strings.LastIndex(job.Title, " — DOU"); idx > 0 {
		job.Title = job.Title[:idx]
	}

	job.Title = cleanText(job.Title)
	job.CompanyName = cleanText(job.CompanyName)
	job.Location = cleanText(job.Location)
	job.Description = cleanText(job.Description)

	if job.Title == "" {
		return nil, fmt.Errorf("%w: no title found", ErrParseFailed)
	}

	return job, nil
}

// findElementText finds the text content of an element by tag name and class
func findElementText(n *html.Node, tag, class string) string {
	if n.Type == html.ElementNode && n.Data == tag {
		for _, attr := range n.Attr {
			if attr.Key == "class" && containsClass(attr.Val, class) {
				return extractText(n)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElementText(c, tag, class); result != "" {
			return result
		}
	}

	return ""
}

// containsClass checks if a class string contains a specific class
func containsClass(classes, target string) bool {
	for _, c := range strings.Fields(classes) {
		if c == target {
			return true
		}
	}
	return false
}

// extractText recursively extracts all text content from a node
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}
