package docx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCoverLetterDOCX_ReturnsNonEmptyBytes(t *testing.T) {
	svc := NewDOCXService()

	data := &CoverLetterDOCXData{
		RecipientName:  "Jane Smith",
		RecipientTitle: "Hiring Manager",
		CompanyName:    "Acme Corp",
		CompanyAddress: "123 Main St, New York, NY",
		Date:           "January 15, 2026",
		Greeting:       "Dear Hiring Manager,",
		Paragraphs:     []string{"I am writing to apply.", "I look forward to hearing from you."},
		Closing:        "Sincerely,",
	}

	result, err := svc.GenerateCoverLetterDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0, "DOCX output should not be empty")
}

func TestGenerateCoverLetterDOCX_HandlesEmptyFields(t *testing.T) {
	svc := NewDOCXService()

	data := &CoverLetterDOCXData{
		RecipientName:  "",
		RecipientTitle: "",
		CompanyName:    "",
		CompanyAddress: "",
		Date:           "",
		Greeting:       "",
		Paragraphs:     []string{},
		Closing:        "",
	}

	result, err := svc.GenerateCoverLetterDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0, "DOCX output should not be empty even with empty fields")
}

func TestGenerateCoverLetterDOCX_HandlesNilParagraphs(t *testing.T) {
	svc := NewDOCXService()

	data := &CoverLetterDOCXData{
		RecipientName: "John Doe",
		Paragraphs:   nil,
	}

	result, err := svc.GenerateCoverLetterDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0)
}

func TestGenerateCoverLetterDOCX_HandlesEmptyParagraphStrings(t *testing.T) {
	svc := NewDOCXService()

	data := &CoverLetterDOCXData{
		RecipientName: "John Doe",
		Paragraphs:   []string{"", "Non-empty paragraph", ""},
		Greeting:      "Dear John,",
		Closing:       "Best,",
	}

	result, err := svc.GenerateCoverLetterDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0)
}

func TestBuildRecipientLine(t *testing.T) {
	tests := []struct {
		name     string
		rName    string
		rTitle   string
		company  string
		expected string
	}{
		{
			name:     "all fields present",
			rName:    "Jane Smith",
			rTitle:   "Hiring Manager",
			company:  "Acme Corp",
			expected: "Jane Smith, Hiring Manager, Acme Corp",
		},
		{
			name:     "name only",
			rName:    "Jane Smith",
			rTitle:   "",
			company:  "",
			expected: "Jane Smith",
		},
		{
			name:     "title only",
			rName:    "",
			rTitle:   "Hiring Manager",
			company:  "",
			expected: "Hiring Manager",
		},
		{
			name:     "company only",
			rName:    "",
			rTitle:   "",
			company:  "Acme Corp",
			expected: "Acme Corp",
		},
		{
			name:     "name and company",
			rName:    "Jane Smith",
			rTitle:   "",
			company:  "Acme Corp",
			expected: "Jane Smith, Acme Corp",
		},
		{
			name:     "all empty",
			rName:    "",
			rTitle:   "",
			company:  "",
			expected: "",
		},
		{
			name:     "title and company",
			rName:    "",
			rTitle:   "CTO",
			company:  "Big Corp",
			expected: "CTO, Big Corp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRecipientLine(tt.rName, tt.rTitle, tt.company)
			assert.Equal(t, tt.expected, result)
		})
	}
}
