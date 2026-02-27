package parser

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// IndeedParser parses job postings from Indeed
type IndeedParser struct {
	fetcher *Fetcher
}

// NewIndeedParser creates a new Indeed parser
func NewIndeedParser(fetcher *Fetcher) *IndeedParser {
	return &IndeedParser{fetcher: fetcher}
}

// CanParse checks if the URL is an Indeed job posting
func (p *IndeedParser) CanParse(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "www.indeed.com" || host == "indeed.com" ||
		strings.HasSuffix(host, ".indeed.com")
}

// Parse extracts job data from an Indeed URL
func (p *IndeedParser) Parse(ctx context.Context, url string) (*ParsedJob, error) {
	body, err := p.fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, err
	}

	// Try JSON-LD first
	if ld, ok := ExtractJobPostingLD(body); ok {
		return &ParsedJob{
			Title:       cleanText(ld.GetTitle()),
			CompanyName: cleanText(ld.GetCompanyName()),
			Location:    cleanText(ld.GetLocation()),
			Description: cleanText(ld.Description),
			Source:      "indeed",
			URL:         url,
		}, nil
	}

	// Fallback: HTML meta tags
	return p.parseHTML(body, url)
}

func (p *IndeedParser) parseHTML(body []byte, url string) (*ParsedJob, error) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	job := &ParsedJob{
		Source: "indeed",
		URL:    url,
	}

	job.Title = findMeta(doc, "og:title")
	if job.Title == "" {
		job.Title = findTitle(doc)
	}
	job.Description = findMeta(doc, "og:description")

	// Indeed title format: "Title - Company - Location | Indeed.com"
	if strings.Contains(job.Title, " | Indeed") {
		job.Title = strings.Split(job.Title, " | Indeed")[0]
	}

	parts := strings.SplitN(job.Title, " - ", 3)
	if len(parts) >= 2 {
		job.Title = strings.TrimSpace(parts[0])
		job.CompanyName = strings.TrimSpace(parts[1])
		if len(parts) == 3 {
			job.Location = strings.TrimSpace(parts[2])
		}
	}

	if job.Title == "" {
		return nil, fmt.Errorf("%w: no title found", ErrParseFailed)
	}

	return job, nil
}
