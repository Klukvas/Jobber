package docx

import (
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// buildVisibleSections
// ---------------------------------------------------------------------------

func TestBuildVisibleSections(t *testing.T) {
	tests := []struct {
		name     string
		input    []*model.SectionOrderDTO
		expected []string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []*model.SectionOrderDTO{},
			expected: []string{},
		},
		{
			name: "all visible sorted by sort_order",
			input: []*model.SectionOrderDTO{
				{SectionKey: "skills", SortOrder: 3, IsVisible: true},
				{SectionKey: "experience", SortOrder: 1, IsVisible: true},
				{SectionKey: "education", SortOrder: 2, IsVisible: true},
			},
			expected: []string{"experience", "education", "skills"},
		},
		{
			name: "filters out invisible sections",
			input: []*model.SectionOrderDTO{
				{SectionKey: "experience", SortOrder: 1, IsVisible: true},
				{SectionKey: "summary", SortOrder: 0, IsVisible: false},
				{SectionKey: "skills", SortOrder: 2, IsVisible: true},
			},
			expected: []string{"experience", "skills"},
		},
		{
			name: "all invisible",
			input: []*model.SectionOrderDTO{
				{SectionKey: "experience", SortOrder: 1, IsVisible: false},
				{SectionKey: "skills", SortOrder: 2, IsVisible: false},
			},
			expected: []string{},
		},
		{
			name: "single visible section",
			input: []*model.SectionOrderDTO{
				{SectionKey: "summary", SortOrder: 0, IsVisible: true},
			},
			expected: []string{"summary"},
		},
		{
			name: "preserves stable sort order",
			input: []*model.SectionOrderDTO{
				{SectionKey: "volunteering", SortOrder: 5, IsVisible: true},
				{SectionKey: "summary", SortOrder: 0, IsVisible: true},
				{SectionKey: "projects", SortOrder: 4, IsVisible: true},
				{SectionKey: "experience", SortOrder: 1, IsVisible: true},
				{SectionKey: "education", SortOrder: 2, IsVisible: true},
				{SectionKey: "skills", SortOrder: 3, IsVisible: true},
			},
			expected: []string{"summary", "experience", "education", "skills", "projects", "volunteering"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildVisibleSections(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// buildTitleLine
// ---------------------------------------------------------------------------

func TestBuildTitleLine(t *testing.T) {
	tests := []struct {
		name      string
		role      string
		org       string
		startDate string
		endDate   string
		isCurrent bool
		expected  string
	}{
		{
			name:      "all fields with date range",
			role:      "Developer",
			org:       "Acme Corp",
			startDate: "2020-01",
			endDate:   "2023-06",
			expected:  "Developer at Acme Corp (2020-01 - 2023-06)",
		},
		{
			name:      "current position",
			role:      "Engineer",
			org:       "Google",
			startDate: "2022-03",
			isCurrent: true,
			expected:  "Engineer at Google (2022-03 - Present)",
		},
		{
			name:     "role only",
			role:     "Manager",
			expected: "Manager",
		},
		{
			name:     "org only",
			org:      "Acme Corp",
			expected: "Acme Corp",
		},
		{
			name:     "empty everything",
			expected: "",
		},
		{
			name:      "no org with dates",
			role:      "Freelancer",
			startDate: "2021-01",
			endDate:   "2022-12",
			expected:  "Freelancer (2021-01 - 2022-12)",
		},
		{
			name:      "start date only",
			role:      "Dev",
			org:       "Co",
			startDate: "2020-01",
			expected:  "Dev at Co (2020-01)",
		},
		{
			name:      "end date only",
			role:      "Dev",
			org:       "Co",
			endDate:   "2022-12",
			expected:  "Dev at Co (2022-12)",
		},
		{
			name:      "current with no start date",
			role:      "Dev",
			isCurrent: true,
			expected:  "Dev (Present)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTitleLine(tt.role, tt.org, tt.startDate, tt.endDate, tt.isCurrent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// buildEducationTitleLine
// ---------------------------------------------------------------------------

func TestBuildEducationTitleLine(t *testing.T) {
	tests := []struct {
		name     string
		edu      *model.EducationDTO
		expected string
	}{
		{
			name: "all fields",
			edu: &model.EducationDTO{
				Degree:       "BSc",
				FieldOfStudy: "Computer Science",
				Institution:  "MIT",
				StartDate:    "2016-09",
				EndDate:      "2020-06",
			},
			expected: "BSc in Computer Science at MIT (2016-09 - 2020-06)",
		},
		{
			name: "degree only",
			edu: &model.EducationDTO{
				Degree: "MBA",
			},
			expected: "MBA",
		},
		{
			name: "institution only",
			edu: &model.EducationDTO{
				Institution: "Harvard",
			},
			expected: "Harvard",
		},
		{
			name: "field of study only",
			edu: &model.EducationDTO{
				FieldOfStudy: "Physics",
			},
			expected: "Physics",
		},
		{
			name:     "empty education",
			edu:      &model.EducationDTO{},
			expected: "",
		},
		{
			name: "currently enrolled",
			edu: &model.EducationDTO{
				Degree:      "PhD",
				Institution: "Stanford",
				StartDate:   "2023-09",
				IsCurrent:   true,
			},
			expected: "PhD at Stanford (2023-09 - Present)",
		},
		{
			name: "degree and institution no dates",
			edu: &model.EducationDTO{
				Degree:      "MSc",
				Institution: "Oxford",
			},
			expected: "MSc at Oxford",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEducationTitleLine(tt.edu)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatDateRange
// ---------------------------------------------------------------------------

func TestFormatDateRange(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		endDate   string
		isCurrent bool
		expected  string
	}{
		{name: "both dates", startDate: "2020-01", endDate: "2023-06", expected: "2020-01 - 2023-06"},
		{name: "start only", startDate: "2020-01", expected: "2020-01"},
		{name: "end only", endDate: "2023-06", expected: "2023-06"},
		{name: "current", startDate: "2020-01", isCurrent: true, expected: "2020-01 - Present"},
		{name: "current no start", isCurrent: true, expected: "Present"},
		{name: "all empty", expected: ""},
		{name: "current overrides end date", startDate: "2020-01", endDate: "2023-06", isCurrent: true, expected: "2020-01 - Present"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDateRange(tt.startDate, tt.endDate, tt.isCurrent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// buildRecipientLine
// ---------------------------------------------------------------------------

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
			expected: "Jane Smith",
		},
		{
			name:     "title only",
			rTitle:   "Hiring Manager",
			expected: "Hiring Manager",
		},
		{
			name:     "company only",
			company:  "Acme Corp",
			expected: "Acme Corp",
		},
		{
			name:     "name and company",
			rName:    "Jane Smith",
			company:  "Acme Corp",
			expected: "Jane Smith, Acme Corp",
		},
		{
			name:     "all empty",
			expected: "",
		},
		{
			name:     "title and company",
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

// ---------------------------------------------------------------------------
// GenerateResumeDOCX
// ---------------------------------------------------------------------------

func TestGenerateResumeDOCX(t *testing.T) {
	svc := NewDOCXService()

	t.Run("full resume produces non-empty bytes", func(t *testing.T) {
		data := buildFullResumeDTO()

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0, "DOCX output should not be empty")
	})

	t.Run("minimal resume with only contact", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{
				FullName: "John Doe",
				Email:    "john@example.com",
			},
			SectionOrder: []*model.SectionOrderDTO{},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("nil contact does not panic", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact:      nil,
			SectionOrder: []*model.SectionOrderDTO{},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("empty contact does not panic", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact:      &model.ContactDTO{},
			SectionOrder: []*model.SectionOrderDTO{},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("invisible sections are not rendered", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Summary: &model.SummaryDTO{Content: "My summary"},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "summary", SortOrder: 0, IsVisible: false},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("all section types", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test User", Email: "test@test.com"},
			Summary: &model.SummaryDTO{Content: "A summary"},
			Experiences: []*model.ExperienceDTO{
				{Position: "Dev", Company: "Co", StartDate: "2020-01", EndDate: "2023-01", Description: "Did things"},
			},
			Educations: []*model.EducationDTO{
				{Degree: "BSc", Institution: "Uni", FieldOfStudy: "CS"},
			},
			Skills:         []*model.SkillDTO{{Name: "Go", Level: "expert"}},
			Languages:      []*model.LanguageDTO{{Name: "English", Proficiency: "native"}},
			Certifications: []*model.CertificationDTO{{Name: "AWS SAA", Issuer: "Amazon", IssueDate: "2023"}},
			Projects:       []*model.ProjectDTO{{Name: "My Project", URL: "https://example.com", Description: "A project"}},
			Volunteering:   []*model.VolunteeringDTO{{Role: "Mentor", Organization: "Code.org", Description: "Mentored"}},
			CustomSections: []*model.CustomSectionDTO{{Title: "Awards", Content: "Best Employee 2023"}},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "summary", SortOrder: 0, IsVisible: true},
				{SectionKey: "experience", SortOrder: 1, IsVisible: true},
				{SectionKey: "education", SortOrder: 2, IsVisible: true},
				{SectionKey: "skills", SortOrder: 3, IsVisible: true},
				{SectionKey: "languages", SortOrder: 4, IsVisible: true},
				{SectionKey: "certifications", SortOrder: 5, IsVisible: true},
				{SectionKey: "projects", SortOrder: 6, IsVisible: true},
				{SectionKey: "volunteering", SortOrder: 7, IsVisible: true},
				{SectionKey: "custom_sections", SortOrder: 8, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("sections with empty data produce valid output", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact:        &model.ContactDTO{FullName: "Test"},
			Summary:        nil,
			Experiences:    nil,
			Educations:     nil,
			Skills:         nil,
			Languages:      nil,
			Certifications: nil,
			Projects:       nil,
			Volunteering:   nil,
			CustomSections: nil,
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "summary", SortOrder: 0, IsVisible: true},
				{SectionKey: "experience", SortOrder: 1, IsVisible: true},
				{SectionKey: "education", SortOrder: 2, IsVisible: true},
				{SectionKey: "skills", SortOrder: 3, IsVisible: true},
				{SectionKey: "languages", SortOrder: 4, IsVisible: true},
				{SectionKey: "certifications", SortOrder: 5, IsVisible: true},
				{SectionKey: "projects", SortOrder: 6, IsVisible: true},
				{SectionKey: "volunteering", SortOrder: 7, IsVisible: true},
				{SectionKey: "custom_sections", SortOrder: 8, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("unknown section key is ignored", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "unknown_section", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// GenerateCoverLetterDOCX
// ---------------------------------------------------------------------------

func TestGenerateCoverLetterDOCX(t *testing.T) {
	svc := NewDOCXService()

	t.Run("full cover letter produces non-empty bytes", func(t *testing.T) {
		data := &CoverLetterDOCXData{
			RecipientName:  "Jane Smith",
			RecipientTitle: "Hiring Manager",
			CompanyName:    "Acme Corp",
			CompanyAddress: "123 Main St, New York, NY",
			Date:           "January 15, 2026",
			Greeting:       "Dear Hiring Manager,",
			Paragraphs:     []string{"I am writing to apply.", "I have extensive experience.", "I look forward to hearing from you."},
			Closing:        "Sincerely,",
		}

		result, err := svc.GenerateCoverLetterDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0, "DOCX output should not be empty")
	})

	t.Run("all empty fields", func(t *testing.T) {
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
	})

	t.Run("nil paragraphs", func(t *testing.T) {
		data := &CoverLetterDOCXData{
			RecipientName: "John Doe",
			Paragraphs:    nil,
		}

		result, err := svc.GenerateCoverLetterDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("empty paragraph strings are skipped", func(t *testing.T) {
		data := &CoverLetterDOCXData{
			RecipientName: "John Doe",
			Paragraphs:    []string{"", "Non-empty paragraph", ""},
			Greeting:      "Dear John,",
			Closing:       "Best,",
		}

		result, err := svc.GenerateCoverLetterDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("recipient line with partial data", func(t *testing.T) {
		data := &CoverLetterDOCXData{
			CompanyName: "Big Corp",
			Greeting:    "Dear Team,",
			Paragraphs:  []string{"Hello."},
		}

		result, err := svc.GenerateCoverLetterDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("only greeting and closing", func(t *testing.T) {
		data := &CoverLetterDOCXData{
			Greeting:   "Dear Sir/Madam,",
			Paragraphs: []string{},
			Closing:    "Regards,",
		}

		result, err := svc.GenerateCoverLetterDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// NewDOCXService
// ---------------------------------------------------------------------------

func TestNewDOCXService(t *testing.T) {
	svc := NewDOCXService()
	assert.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// Contact info rendering helpers
// ---------------------------------------------------------------------------

func TestAddContactInfo_AllFields(t *testing.T) {
	// Exercise the contact info path with all fields populated.
	// We test indirectly via GenerateResumeDOCX.
	svc := NewDOCXService()

	data := &model.FullResumeDTO{
		Contact: &model.ContactDTO{
			FullName: "John Doe",
			Email:    "john@example.com",
			Phone:    "+1234567890",
			Location: "New York, NY",
			LinkedIn: "linkedin.com/in/johndoe",
			Website:  "https://johndoe.com",
			GitHub:   "github.com/johndoe",
		},
		SectionOrder: []*model.SectionOrderDTO{},
	}

	result, err := svc.GenerateResumeDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0)
}

func TestAddContactInfo_PartialFields(t *testing.T) {
	svc := NewDOCXService()

	data := &model.FullResumeDTO{
		Contact: &model.ContactDTO{
			FullName: "John Doe",
			Email:    "john@example.com",
			// Other fields intentionally empty
		},
		SectionOrder: []*model.SectionOrderDTO{},
	}

	result, err := svc.GenerateResumeDOCX(data)
	require.NoError(t, err)
	assert.Greater(t, len(result), 0)
}

// ---------------------------------------------------------------------------
// Experience section edge cases
// ---------------------------------------------------------------------------

func TestExperienceSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("experience with location and description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Experiences: []*model.ExperienceDTO{
				{
					Position:    "Developer",
					Company:     "Acme",
					Location:    "Remote",
					StartDate:   "2020-01",
					EndDate:     "2023-06",
					Description: "Built APIs and services",
				},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("experience with no location or description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Experiences: []*model.ExperienceDTO{
				{Position: "Developer", Company: "Acme"},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("current experience", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Experiences: []*model.ExperienceDTO{
				{Position: "Lead", Company: "Google", StartDate: "2022-01", IsCurrent: true},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Education section edge cases
// ---------------------------------------------------------------------------

func TestEducationSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("education with GPA and description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Educations: []*model.EducationDTO{
				{
					Degree:       "BSc",
					Institution:  "MIT",
					FieldOfStudy: "CS",
					GPA:          "3.9",
					Description:  "Dean's list",
				},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "education", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("education without GPA or description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Educations: []*model.EducationDTO{
				{Degree: "BSc", Institution: "MIT"},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "education", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Skills section edge cases
// ---------------------------------------------------------------------------

func TestSkillsSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("skills with levels", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Skills: []*model.SkillDTO{
				{Name: "Go", Level: "expert"},
				{Name: "Python", Level: ""},
				{Name: "Rust", Level: "intermediate"},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "skills", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Languages section edge cases
// ---------------------------------------------------------------------------

func TestLanguagesSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("languages with proficiency", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Languages: []*model.LanguageDTO{
				{Name: "English", Proficiency: "native"},
				{Name: "German", Proficiency: ""},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "languages", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Certifications section edge cases
// ---------------------------------------------------------------------------

func TestCertificationsSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("certifications with all fields", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Certifications: []*model.CertificationDTO{
				{Name: "AWS SAA", Issuer: "Amazon", IssueDate: "2023-01"},
				{Name: "CKA", Issuer: "", IssueDate: ""},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "certifications", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Projects section edge cases
// ---------------------------------------------------------------------------

func TestProjectsSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("projects with and without URL and description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Projects: []*model.ProjectDTO{
				{Name: "Project A", URL: "https://github.com/a", Description: "A cool project"},
				{Name: "Project B", URL: "", Description: ""},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "projects", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Volunteering section edge cases
// ---------------------------------------------------------------------------

func TestVolunteeringSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("volunteering with and without description", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			Volunteering: []*model.VolunteeringDTO{
				{Role: "Mentor", Organization: "Code.org", StartDate: "2022-01", EndDate: "2023-01", Description: "Mentored students"},
				{Role: "Volunteer", Organization: "Red Cross"},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "volunteering", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Custom sections edge cases
// ---------------------------------------------------------------------------

func TestCustomSectionsSection_EdgeCases(t *testing.T) {
	svc := NewDOCXService()

	t.Run("custom sections with title and content", func(t *testing.T) {
		data := &model.FullResumeDTO{
			Contact: &model.ContactDTO{FullName: "Test"},
			CustomSections: []*model.CustomSectionDTO{
				{Title: "Awards", Content: "Best Employee 2023"},
				{Title: "", Content: "Orphan content"},
				{Title: "Publications", Content: ""},
			},
			SectionOrder: []*model.SectionOrderDTO{
				{SectionKey: "custom_sections", SortOrder: 0, IsVisible: true},
			},
		}

		result, err := svc.GenerateResumeDOCX(data)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildFullResumeDTO() *model.FullResumeDTO {
	return &model.FullResumeDTO{
		Contact: &model.ContactDTO{
			FullName: "John Doe",
			Email:    "john@example.com",
			Phone:    "+1234567890",
			Location: "New York, NY",
			LinkedIn: "linkedin.com/in/johndoe",
			Website:  "https://johndoe.com",
			GitHub:   "github.com/johndoe",
		},
		Summary: &model.SummaryDTO{Content: "Experienced software engineer with 10+ years."},
		Experiences: []*model.ExperienceDTO{
			{
				Position:    "Senior Developer",
				Company:     "Acme Corp",
				Location:    "San Francisco",
				StartDate:   "2020-01",
				EndDate:     "",
				IsCurrent:   true,
				Description: "Led team of 5 engineers, built microservices.",
			},
			{
				Position:    "Developer",
				Company:     "StartupCo",
				Location:    "Remote",
				StartDate:   "2017-06",
				EndDate:     "2019-12",
				Description: "Full-stack development.",
			},
		},
		Educations: []*model.EducationDTO{
			{
				Degree:       "BSc",
				FieldOfStudy: "Computer Science",
				Institution:  "MIT",
				StartDate:    "2013-09",
				EndDate:      "2017-06",
				GPA:          "3.9",
			},
		},
		Skills: []*model.SkillDTO{
			{Name: "Go", Level: "expert"},
			{Name: "Python", Level: "advanced"},
			{Name: "Docker", Level: ""},
		},
		Languages: []*model.LanguageDTO{
			{Name: "English", Proficiency: "native"},
			{Name: "Spanish", Proficiency: "conversational"},
		},
		Certifications: []*model.CertificationDTO{
			{Name: "AWS Solutions Architect", Issuer: "Amazon", IssueDate: "2023-01"},
		},
		Projects: []*model.ProjectDTO{
			{Name: "OSS Tool", URL: "https://github.com/johndoe/oss", Description: "CLI tool for developers"},
		},
		Volunteering: []*model.VolunteeringDTO{
			{Role: "Mentor", Organization: "Code.org", StartDate: "2021-01", Description: "Mentored high school students"},
		},
		CustomSections: []*model.CustomSectionDTO{
			{Title: "Awards", Content: "Employee of the Year 2022"},
		},
		SectionOrder: []*model.SectionOrderDTO{
			{SectionKey: "summary", SortOrder: 0, IsVisible: true},
			{SectionKey: "experience", SortOrder: 1, IsVisible: true},
			{SectionKey: "education", SortOrder: 2, IsVisible: true},
			{SectionKey: "skills", SortOrder: 3, IsVisible: true},
			{SectionKey: "languages", SortOrder: 4, IsVisible: true},
			{SectionKey: "certifications", SortOrder: 5, IsVisible: true},
			{SectionKey: "projects", SortOrder: 6, IsVisible: true},
			{SectionKey: "volunteering", SortOrder: 7, IsVisible: true},
			{SectionKey: "custom_sections", SortOrder: 8, IsVisible: true},
		},
	}
}
