package parser

import (
	"context"
	"errors"
)

// ParsedJob represents structured job data extracted from a URL
type ParsedJob struct {
	Title       string `json:"title"`
	CompanyName string `json:"company_name,omitempty"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
	URL         string `json:"url"`
}

// Parser defines the interface for job parsers
type Parser interface {
	CanParse(url string) bool
	Parse(ctx context.Context, url string) (*ParsedJob, error)
}

// Sentinel errors
var (
	ErrUnsupportedJobSite = errors.New("unsupported job site")
	ErrFetchFailed        = errors.New("failed to fetch URL")
	ErrParseFailed        = errors.New("failed to parse job data")
)

// Registry holds all parsers and picks the right one by URL
type Registry struct {
	parsers []Parser
}

// NewRegistry creates a new parser registry with all available parsers
func NewRegistry(fetcher *Fetcher) *Registry {
	return &Registry{
		parsers: []Parser{
			NewLinkedInParser(fetcher),
			NewIndeedParser(fetcher),
			NewDOUParser(fetcher),
		},
	}
}

// Parse finds the appropriate parser for the URL and extracts job data
func (r *Registry) Parse(ctx context.Context, url string) (*ParsedJob, error) {
	for _, p := range r.parsers {
		if p.CanParse(url) {
			return p.Parse(ctx, url)
		}
	}
	return nil, ErrUnsupportedJobSite
}
