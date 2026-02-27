package parser

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// LinkedInParser parses job postings from LinkedIn
type LinkedInParser struct {
	fetcher *Fetcher
}

// NewLinkedInParser creates a new LinkedIn parser
func NewLinkedInParser(fetcher *Fetcher) *LinkedInParser {
	return &LinkedInParser{fetcher: fetcher}
}

// CanParse checks if the URL is a LinkedIn job posting
func (p *LinkedInParser) CanParse(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return (host == "www.linkedin.com" || host == "linkedin.com") &&
		strings.HasPrefix(u.Path, "/jobs")
}

// Parse extracts job data from a LinkedIn URL
func (p *LinkedInParser) Parse(ctx context.Context, url string) (*ParsedJob, error) {
	body, err := p.fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}

	// Try JSON-LD first (most reliable for LinkedIn)
	if ld, ok := ExtractJobPostingLD(body); ok {
		return &ParsedJob{
			Title:       cleanText(ld.GetTitle()),
			CompanyName: cleanText(ld.GetCompanyName()),
			Location:    cleanText(ld.GetLocation()),
			Description: cleanText(ld.Description),
			Source:      "linkedin",
			URL:         url,
		}, nil
	}

	// Fallback: parse HTML meta tags and content
	return p.parseHTML(body, url)
}

func (p *LinkedInParser) parseHTML(body []byte, url string) (*ParsedJob, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	job := &ParsedJob{
		Source: "linkedin",
		URL:    url,
	}

	// Extract from <meta> tags
	job.Title = findMeta(doc, "og:title")
	if job.Title == "" {
		job.Title = findTitle(doc)
	}
	job.Description = findMeta(doc, "og:description")

	// Clean up LinkedIn title format: "Title at Company | LinkedIn"
	if strings.Contains(job.Title, " | LinkedIn") {
		job.Title = strings.TrimSuffix(job.Title, " | LinkedIn")
	}

	// Try to extract company from title pattern: "Job Title at Company"
	if parts := strings.SplitN(job.Title, " at ", 2); len(parts) == 2 {
		job.Title = strings.TrimSpace(parts[0])
		job.CompanyName = strings.TrimSpace(parts[1])
	}

	if job.Title == "" {
		return nil, fmt.Errorf("%w: no title found", ErrParseFailed)
	}

	return job, nil
}

// findMeta finds a meta tag by property or name
func findMeta(n *html.Node, property string) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var prop, content string
		for _, attr := range n.Attr {
			if (attr.Key == "property" || attr.Key == "name") && attr.Val == property {
				prop = attr.Val
			}
			if attr.Key == "content" {
				content = attr.Val
			}
		}
		if prop != "" {
			return content
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findMeta(c, property); result != "" {
			return result
		}
	}

	return ""
}

// findTitle extracts the <title> text
func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return n.FirstChild.Data
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findTitle(c); result != "" {
			return result
		}
	}

	return ""
}

// cleanText strips leading/trailing whitespace and normalizes internal spaces
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	// Collapse multiple spaces/newlines into a single space
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
