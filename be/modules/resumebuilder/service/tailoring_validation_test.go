package service

import (
	"encoding/json"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTailoringSuggestions(t *testing.T) {
	for _, test := range []struct {
		name      string
		change    func(*model.FullResumeDTO, *ai.ResumeTailoringSuggestions)
		wantError bool
		count     int
	}{
		{name: "valid summary", count: 1},
		{name: "invented metric", change: func(_ *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Summary.After = "Delivered 100% improvement"
		}, wantError: true},
		{name: "unsupported quote", change: func(_ *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Summary.Evidence[0].Quote = "invented evidence"
		}, wantError: true},
		{name: "unknown experience", change: func(_ *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Experiences = []ai.TailoringTextSuggestion{{ExperienceID: "foreign", After: "New text"}}
		}, wantError: true},
		{name: "invented skill", change: func(_ *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.SkillOrder = &ai.TailoringSkillSuggestion{SkillIDs: []string{"sk-1", "foreign"}, Reason: "Relevance", Evidence: []ai.TailoringEvidence{{Section: "skills", EntryID: "sk-1", Quote: "Go"}}}
		}, wantError: true},
		{name: "duplicate skill", change: func(_ *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.SkillOrder = &ai.TailoringSkillSuggestion{SkillIDs: []string{"sk-1", "sk-1"}, Reason: "Relevance", Evidence: []ai.TailoringEvidence{{Section: "skills", EntryID: "sk-1", Quote: "Go"}}}
		}, wantError: true},
		{name: "all unchanged", change: func(source *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Summary.After = source.Summary.Content
		}, count: 0},
		{name: "unchanged summary plus experience", change: func(source *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Summary.After = source.Summary.Content
			s.Experiences = []ai.TailoringTextSuggestion{{ExperienceID: source.Experiences[0].ID, After: "Led a backend engineering team of 5.", Reason: "Highlight backend", Evidence: []ai.TailoringEvidence{{Section: "experience", EntryID: source.Experiences[0].ID, Quote: source.Experiences[0].Description}}}}
		}, count: 1},
		{name: "blank company and position keys", change: func(source *model.FullResumeDTO, s *ai.ResumeTailoringSuggestions) {
			s.Summary = nil
			source.Experiences[0].Company = ""
			source.Experiences[0].Position = ""
			s.Experiences = []ai.TailoringTextSuggestion{{ExperienceID: source.Experiences[0].ID, After: "Led a backend engineering team of 5.", Reason: "Highlight backend", Evidence: []ai.TailoringEvidence{{Section: "experience", EntryID: source.Experiences[0].ID, Quote: source.Experiences[0].Description}}}}
		}, count: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, r, _, client, _ := tailoringFixture(t)
			if test.change != nil {
				test.change(r.source, client.result)
			}
			changes, err := validateTailoringSuggestions(r.source, client.result)
			if test.wantError {
				require.ErrorIs(t, err, model.ErrAIInvalidResponse)
				return
			}
			require.NoError(t, err)
			require.Len(t, changes, test.count)
			data, err := json.Marshal(changes)
			require.NoError(t, err)
			if test.count == 0 {
				assert.Equal(t, "[]", string(data))
			}
			if test.name == "blank company and position keys" {
				assert.Contains(t, string(data), `"company":""`)
				assert.Contains(t, string(data), `"position":""`)
			}
		})
	}
}

func TestApplyTailoringChangesPreservesSource(t *testing.T) {
	source := newFullResumeForAI()
	source.SectionOrder = []*model.SectionOrderDTO{{SectionKey: "skills", Column: "sidebar", IsVisible: false, SortOrder: 2}}
	original := hashTailoring(source)
	changes := []model.TailoringChange{{Kind: "summary", After: tailoringJSON("Adapted summary")}, {Kind: "skill_order", After: tailoringJSON([]model.TailoringSkillRef{{ID: "sk-2", Name: "TypeScript"}, {ID: "sk-1", Name: "Go"}})}}
	result, err := applyTailoringChanges(source, changes)
	require.NoError(t, err)
	assert.Equal(t, "Adapted summary", result.Summary.Content)
	assert.Equal(t, "sk-2", result.Skills[0].ID)
	assert.Equal(t, source.Skills[0].Level, result.Skills[1].Level)
	assert.Equal(t, source.SectionOrder, result.SectionOrder)
	assert.Equal(t, original, hashTailoring(source))
}

func TestTailoringHiddenSections(t *testing.T) {
	s, r, _, client, _ := tailoringFixture(t)
	r.source.SectionOrder[1].IsVisible = false
	r.source.Experiences[0].Description = "Private confidential project"
	_, err := s.Generate(t.Context(), "user", "job", r.source.ID, "en")
	require.NoError(t, err)
	assert.NotContains(t, client.facts, "Private confidential project")
	client.result.Summary.Evidence = []ai.TailoringEvidence{{Section: "experience", EntryID: r.source.Experiences[0].ID, Quote: "Private confidential project"}}
	_, err = validateTailoringSuggestions(r.source, client.result)
	assert.ErrorIs(t, err, model.ErrAIInvalidResponse)
	copied, err := applyTailoringChanges(r.source, []model.TailoringChange{{Kind: "summary", After: tailoringJSON("Visible summary")}})
	require.NoError(t, err)
	assert.Equal(t, "Private confidential project", copied.Experiences[0].Description)
	assert.False(t, copied.SectionOrder[1].IsVisible)
}

func TestTailoringMetricPairs(t *testing.T) {
	for _, test := range []struct {
		name, before, after string
		valid               bool
	}{
		{name: "services cannot become percent", before: "Maintained 10 services", after: "Improved performance 10%"},
		{name: "independent percent cannot move", before: "Maintained 10 services and improved performance 20%", after: "Improved performance 10%"},
		{name: "services cannot become currency", before: "Maintained 10 services", after: "Saved $10"},
		{name: "percent preserved", before: "Improved performance 10%", after: "Performance improved by 10 %", valid: true},
		{name: "currency preserved", before: "Saved $ 10", after: "Savings of $10", valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, r, _, client, _ := tailoringFixture(t)
			r.source.Summary.Content = test.before
			client.result.Summary.After = test.after
			client.result.Summary.Evidence[0].Quote = test.before
			_, err := validateTailoringSuggestions(r.source, client.result)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, model.ErrAIInvalidResponse)
			}
		})
	}
}
