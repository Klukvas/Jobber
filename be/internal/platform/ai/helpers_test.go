package ai

import (
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeForLLM(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "normal text",
			text:     "Hello, world!",
			maxLen:   100,
			expected: "<document>\nHello, world!\n</document>",
		},
		{
			name:     "text with null bytes",
			text:     "Hello\x00World\x00!",
			maxLen:   100,
			expected: "<document>\nHelloWorld!\n</document>",
		},
		{
			name:     "text exceeding maxLen is truncated rune-safe",
			text:     "abcdefghij",
			maxLen:   5,
			expected: "<document>\nabcde\n</document>",
		},
		{
			name:     "empty string",
			text:     "",
			maxLen:   100,
			expected: "<document>\n\n</document>",
		},
		{
			name:     "unicode text within limit",
			text:     "Привет мир",
			maxLen:   100,
			expected: "<document>\nПривет мир\n</document>",
		},
		{
			name:     "unicode text truncated at rune boundary",
			text:     "Привет мир",
			maxLen:   6,
			expected: "<document>\nПривет\n</document>",
		},
		{
			name:     "null bytes and truncation combined",
			text:     "ab\x00cd\x00efgh",
			maxLen:   4,
			expected: "<document>\nabcd\n</document>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForLLM(tt.text, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no truncation",
			s:        "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			s:        "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "exceeds maxLen",
			s:        "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "empty string",
			s:        "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "unicode multibyte chars within limit",
			s:        "日本語テスト",
			maxLen:   6,
			expected: "日本語テスト",
		},
		{
			name:     "unicode multibyte chars truncated",
			s:        "日本語テスト",
			maxLen:   3,
			expected: "日本語",
		},
		{
			name:     "emoji truncation",
			s:        "Hello 🌍🌎🌏",
			maxLen:   7,
			expected: "Hello 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.s, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLocaleToLanguage(t *testing.T) {
	tests := []struct {
		name     string
		locale   string
		expected string
	}{
		{name: "Russian", locale: "ru", expected: "Russian"},
		{name: "Ukrainian ua", locale: "ua", expected: "Ukrainian"},
		{name: "Ukrainian uk", locale: "uk", expected: "Ukrainian"},
		{name: "German", locale: "de", expected: "German"},
		{name: "French", locale: "fr", expected: "French"},
		{name: "Spanish", locale: "es", expected: "Spanish"},
		{name: "Portuguese", locale: "pt", expected: "Portuguese"},
		{name: "Italian", locale: "it", expected: "Italian"},
		{name: "Polish", locale: "pl", expected: "Polish"},
		{name: "English explicit", locale: "en", expected: "English"},
		{name: "unknown locale", locale: "ja", expected: "English"},
		{name: "empty string", locale: "", expected: "English"},
		{name: "mixed case RU", locale: "RU", expected: "Russian"},
		{name: "mixed case De", locale: "De", expected: "German"},
		{name: "whitespace padded", locale: "  fr  ", expected: "French"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := localeToLanguage(tt.locale)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClampScore(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected int
	}{
		{name: "negative", score: -10, expected: 0},
		{name: "zero", score: 0, expected: 0},
		{name: "mid range", score: 50, expected: 50},
		{name: "max boundary", score: 100, expected: 100},
		{name: "over 100", score: 150, expected: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampScore(tt.score)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateParsedJob(t *testing.T) {
	t.Run("truncates long title", func(t *testing.T) {
		longTitle := strings.Repeat("A", maxTitleLength+100)
		p := &ParsedJob{Title: longTitle}

		validateParsedJob(p)

		assert.Len(t, []rune(p.Title), maxTitleLength)
	})

	t.Run("truncates long description", func(t *testing.T) {
		longDesc := strings.Repeat("B", maxDescriptionLength+100)
		p := &ParsedJob{
			Title:       "Test",
			Description: &longDesc,
		}

		validateParsedJob(p)

		assert.Len(t, []rune(*p.Description), maxDescriptionLength)
	})

	t.Run("truncates long company name", func(t *testing.T) {
		longCompany := strings.Repeat("C", maxCompanyNameLength+100)
		p := &ParsedJob{
			Title:       "Test",
			CompanyName: &longCompany,
		}

		validateParsedJob(p)

		assert.Len(t, []rune(*p.CompanyName), maxCompanyNameLength)
	})

	t.Run("truncates long source", func(t *testing.T) {
		longSource := strings.Repeat("D", maxSourceLength+100)
		p := &ParsedJob{
			Title:  "Test",
			Source: &longSource,
		}

		validateParsedJob(p)

		assert.Len(t, []rune(*p.Source), maxSourceLength)
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		p := &ParsedJob{
			Title:       "Test Job",
			Description: nil,
			CompanyName: nil,
			Source:      nil,
		}

		validateParsedJob(p)

		assert.Equal(t, "Test Job", p.Title)
		assert.Nil(t, p.Description)
		assert.Nil(t, p.CompanyName)
		assert.Nil(t, p.Source)
	})

	t.Run("leaves short fields unchanged", func(t *testing.T) {
		desc := "A short description"
		company := "Acme"
		source := "LinkedIn"
		p := &ParsedJob{
			Title:       "Engineer",
			Description: &desc,
			CompanyName: &company,
			Source:      &source,
		}

		validateParsedJob(p)

		assert.Equal(t, "Engineer", p.Title)
		assert.Equal(t, "A short description", *p.Description)
		assert.Equal(t, "Acme", *p.CompanyName)
		assert.Equal(t, "LinkedIn", *p.Source)
	})
}

func TestValidateMatchResult(t *testing.T) {
	t.Run("clamps overall score", func(t *testing.T) {
		r := &MatchResult{OverallScore: 150}

		validateMatchResult(r)

		assert.Equal(t, 100, r.OverallScore)
	})

	t.Run("clamps negative overall score", func(t *testing.T) {
		r := &MatchResult{OverallScore: -5}

		validateMatchResult(r)

		assert.Equal(t, 0, r.OverallScore)
	})

	t.Run("truncates summary", func(t *testing.T) {
		r := &MatchResult{
			Summary: strings.Repeat("X", maxSummaryLength+100),
		}

		validateMatchResult(r)

		assert.Len(t, []rune(r.Summary), maxSummaryLength)
	})

	t.Run("limits categories count", func(t *testing.T) {
		cats := make([]MatchCategory, maxCategories+5)
		for i := range cats {
			cats[i] = MatchCategory{Name: "Cat", Score: 50, Details: "ok"}
		}
		r := &MatchResult{Categories: cats}

		validateMatchResult(r)

		assert.Len(t, r.Categories, maxCategories)
	})

	t.Run("clamps category scores and truncates details", func(t *testing.T) {
		r := &MatchResult{
			Categories: []MatchCategory{
				{Name: "A", Score: 200, Details: strings.Repeat("Z", maxCategoryDetailLength+50)},
				{Name: "B", Score: -10, Details: "short"},
			},
		}

		validateMatchResult(r)

		assert.Equal(t, 100, r.Categories[0].Score)
		assert.Len(t, []rune(r.Categories[0].Details), maxCategoryDetailLength)
		assert.Equal(t, 0, r.Categories[1].Score)
		assert.Equal(t, "short", r.Categories[1].Details)
	})

	t.Run("limits missing keywords count", func(t *testing.T) {
		keywords := make([]string, maxMissingKeywords+10)
		for i := range keywords {
			keywords[i] = "kw"
		}
		r := &MatchResult{MissingKeywords: keywords}

		validateMatchResult(r)

		assert.Len(t, r.MissingKeywords, maxMissingKeywords)
	})

	t.Run("limits strengths count", func(t *testing.T) {
		strengths := make([]string, maxStrengths+10)
		for i := range strengths {
			strengths[i] = "strength"
		}
		r := &MatchResult{Strengths: strengths}

		validateMatchResult(r)

		assert.Len(t, r.Strengths, maxStrengths)
	})

	t.Run("valid result unchanged", func(t *testing.T) {
		r := &MatchResult{
			OverallScore: 75,
			Summary:      "Good match",
			Categories: []MatchCategory{
				{Name: "Skills", Score: 80, Details: "Strong match"},
			},
			MissingKeywords: []string{"Go"},
			Strengths:       []string{"Experience"},
		}

		validateMatchResult(r)

		assert.Equal(t, 75, r.OverallScore)
		assert.Equal(t, "Good match", r.Summary)
		assert.Len(t, r.Categories, 1)
		assert.Equal(t, 80, r.Categories[0].Score)
	})
}

func TestExtractTextFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		content  []anthropic.ContentBlockUnion
		expected string
	}{
		{
			name:     "empty content",
			content:  []anthropic.ContentBlockUnion{},
			expected: "",
		},
		{
			name: "single text block",
			content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: `{"title": "Engineer"}`},
			},
			expected: `{"title": "Engineer"}`,
		},
		{
			name: "multiple text blocks concatenated",
			content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: `{"title": `},
				{Type: "text", Text: `"Engineer"}`},
			},
			expected: `{"title": "Engineer"}`,
		},
		{
			name: "ignores non-text blocks",
			content: []anthropic.ContentBlockUnion{
				{Type: "thinking", Text: ""},
				{Type: "text", Text: `{"score": 90}`},
			},
			expected: `{"score": 90}`,
		},
		{
			name: "strips markdown code fences",
			content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: "```json\n{\"title\": \"Dev\"}\n```"},
			},
			expected: `{"title": "Dev"}`,
		},
		{
			name: "extracts JSON object from surrounding text",
			content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: `Here is the result: {"score": 42} end`},
			},
			expected: `{"score": 42}`,
		},
		{
			name: "text starting with brace returned as is",
			content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: `{"key": "value"}`},
			},
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromResponse(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}
