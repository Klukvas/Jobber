package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestExtractJobPostingLD(t *testing.T) {
	t.Run("extracts JobPosting from JSON-LD", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			{
				"@context": "https://schema.org",
				"@type": "JobPosting",
				"title": "Senior Software Engineer",
				"description": "Build amazing things",
				"hiringOrganization": {
					"@type": "Organization",
					"name": "Acme Corp"
				},
				"jobLocation": {
					"@type": "Place",
					"address": {
						"@type": "PostalAddress",
						"addressLocality": "Berlin",
						"addressRegion": "BE",
						"addressCountry": "Germany"
					}
				}
			}
			</script>
		</head><body></body></html>`)

		ld, ok := ExtractJobPostingLD(input)
		require.True(t, ok)
		assert.Equal(t, "Senior Software Engineer", ld.GetTitle())
		assert.Equal(t, "Acme Corp", ld.GetCompanyName())
		assert.Equal(t, "Berlin, BE, Germany", ld.GetLocation())
	})

	t.Run("extracts from array of JSON-LD objects", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			[
				{"@type": "WebSite", "name": "Example"},
				{
					"@type": "JobPosting",
					"title": "Product Manager",
					"hiringOrganization": "Google"
				}
			]
			</script>
		</head><body></body></html>`)

		ld, ok := ExtractJobPostingLD(input)
		require.True(t, ok)
		assert.Equal(t, "Product Manager", ld.GetTitle())
		assert.Equal(t, "Google", ld.GetCompanyName())
	})

	t.Run("extracts from @graph pattern", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			{
				"@graph": [
					{"@type": "BreadcrumbList"},
					{
						"@type": "JobPosting",
						"title": "DevOps Engineer",
						"name": "DevOps Alternative"
					}
				]
			}
			</script>
		</head><body></body></html>`)

		ld, ok := ExtractJobPostingLD(input)
		require.True(t, ok)
		assert.Equal(t, "DevOps Engineer", ld.GetTitle())
	})

	t.Run("falls back to name when title is empty", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			{
				"@type": "JobPosting",
				"name": "Backend Developer"
			}
			</script>
		</head><body></body></html>`)

		ld, ok := ExtractJobPostingLD(input)
		require.True(t, ok)
		assert.Equal(t, "Backend Developer", ld.GetTitle())
	})

	t.Run("returns false when no JobPosting found", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			{"@type": "WebSite", "name": "Example"}
			</script>
		</head><body></body></html>`)

		_, ok := ExtractJobPostingLD(input)
		assert.False(t, ok)
	})

	t.Run("returns false for empty HTML", func(t *testing.T) {
		_, ok := ExtractJobPostingLD([]byte(`<html><body></body></html>`))
		assert.False(t, ok)
	})

	t.Run("handles location as array", func(t *testing.T) {
		input := []byte(`<html><head>
			<script type="application/ld+json">
			{
				"@type": "JobPosting",
				"title": "Test",
				"jobLocation": [
					{
						"@type": "Place",
						"name": "Remote"
					}
				]
			}
			</script>
		</head><body></body></html>`)

		ld, ok := ExtractJobPostingLD(input)
		require.True(t, ok)
		assert.Equal(t, "Remote", ld.GetLocation())
	})
}

func TestRegistry_CanParse(t *testing.T) {
	fetcher := NewFetcher()
	registry := NewRegistry(fetcher)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"LinkedIn job URL", "https://www.linkedin.com/jobs/view/123", true},
		{"Indeed job URL", "https://www.indeed.com/viewjob?jk=abc", true},
		{"DOU job URL", "https://jobs.dou.ua/companies/test/vacancies/123/", true},
		{"Unsupported site", "https://www.google.com/search?q=jobs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, p := range registry.parsers {
				if p.CanParse(tt.url) {
					found = true
					break
				}
			}
			assert.Equal(t, tt.expected, found)
		})
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  world  ", "hello world"},
		{"\n\tline1\n\tline2\n", "line1 line2"},
		{"", ""},
		{"no extra spaces", "no extra spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, cleanText(tt.input))
		})
	}
}

func TestLinkedInParser_CanParse(t *testing.T) {
	p := NewLinkedInParser(nil)
	assert.True(t, p.CanParse("https://www.linkedin.com/jobs/view/123456"))
	assert.False(t, p.CanParse("https://www.indeed.com/viewjob"))
}

func TestIndeedParser_CanParse(t *testing.T) {
	p := NewIndeedParser(nil)
	assert.True(t, p.CanParse("https://www.indeed.com/viewjob?jk=abc"))
	assert.False(t, p.CanParse("https://www.linkedin.com/jobs"))
}

func TestDOUParser_CanParse(t *testing.T) {
	p := NewDOUParser(nil)
	assert.True(t, p.CanParse("https://jobs.dou.ua/companies/test/vacancies/123/"))
	assert.False(t, p.CanParse("https://www.linkedin.com"))
}

func TestFindElementText(t *testing.T) {
	t.Run("finds element by tag and class", func(t *testing.T) {
		s := `<html><body><h1 class="g-h2">Senior Developer</h1></body></html>`
		doc, err := html.Parse(strings.NewReader(s))
		require.NoError(t, err)
		result := findElementText(doc, "h1", "g-h2")
		assert.Equal(t, "Senior Developer", result)
	})

	t.Run("returns empty when not found", func(t *testing.T) {
		s := `<html><body><h1>Title</h1></body></html>`
		doc, err := html.Parse(strings.NewReader(s))
		require.NoError(t, err)
		result := findElementText(doc, "h1", "special-class")
		assert.Equal(t, "", result)
	})
}

func TestContainsClass(t *testing.T) {
	assert.True(t, containsClass("foo bar baz", "bar"))
	assert.False(t, containsClass("foo bar baz", "ba"))
	assert.True(t, containsClass("single", "single"))
}

func TestFindMeta(t *testing.T) {
	t.Run("finds meta by property", func(t *testing.T) {
		s := `<html><head><meta property="og:title" content="Test Title" /></head><body></body></html>`
		doc, err := html.Parse(strings.NewReader(s))
		require.NoError(t, err)
		assert.Equal(t, "Test Title", findMeta(doc, "og:title"))
	})

	t.Run("returns empty when not found", func(t *testing.T) {
		s := `<html><head></head><body></body></html>`
		doc, err := html.Parse(strings.NewReader(s))
		require.NoError(t, err)
		assert.Equal(t, "", findMeta(doc, "og:title"))
	})
}
