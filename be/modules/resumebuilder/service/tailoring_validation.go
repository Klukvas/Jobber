package service

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/google/uuid"
)

var errNoTailoringChange = errors.New("unchanged tailoring suggestion")

var tailoringNumbers = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)?`)
var tailoringMetricPairs = regexp.MustCompile(`[\p{Sc}]\s*[+-]?[0-9]+(?:[.,][0-9]+)?|[+-]?[0-9]+(?:[.,][0-9]+)?\s*[%‰‱\p{Sc}]`)

func validateTailoringSuggestions(source *model.FullResumeDTO, suggestions *ai.ResumeTailoringSuggestions) ([]model.TailoringChange, error) {
	if suggestions != nil && suggestions.Summary != nil && !tailoringVisible(source, "summary") {
		return nil, model.ErrAIInvalidResponse
	}
	source = visibleTailoringSource(source)
	if suggestions == nil || len(suggestions.Experiences) > len(source.Experiences) {
		return nil, model.ErrAIInvalidResponse
	}
	changes := make([]model.TailoringChange, 0, len(suggestions.Experiences)+2)
	if suggestion := suggestions.Summary; suggestion != nil {
		before := ""
		if source.Summary != nil {
			before = source.Summary.Content
		}
		change, err := validateTailoringText(source, "summary", "", before, suggestion, 5000)
		if err != nil && !errors.Is(err, errNoTailoringChange) {
			return nil, err
		}
		if err == nil {
			changes = append(changes, change)
		}
	}
	byID := make(map[string]*model.ExperienceDTO, len(source.Experiences))
	for _, exp := range source.Experiences {
		byID[exp.ID] = exp
	}
	seen := make(map[string]bool)
	for _, suggestion := range suggestions.Experiences {
		exp, ok := byID[suggestion.ExperienceID]
		if !ok || seen[exp.ID] {
			return nil, model.ErrAIInvalidResponse
		}
		seen[exp.ID] = true
		change, err := validateTailoringText(source, "experience", exp.ID, exp.Description, &suggestion, 10000)
		if errors.Is(err, errNoTailoringChange) {
			continue
		}
		if err != nil {
			return nil, err
		}
		change.ExperienceID = exp.ID
		change.Company = exp.Company
		change.Position = exp.Position
		changes = append(changes, change)
	}
	if suggestion := suggestions.SkillOrder; suggestion != nil {
		change, err := validateTailoringSkillOrder(source, suggestion)
		if err != nil && !errors.Is(err, errNoTailoringChange) {
			return nil, err
		}
		if err == nil {
			changes = append(changes, change)
		}
	}
	return changes, nil
}

func validateTailoringText(source *model.FullResumeDTO, kind, entryID, before string, suggestion *ai.TailoringTextSuggestion, maxLen int) (model.TailoringChange, error) {
	after := strings.TrimSpace(suggestion.After)
	if after == strings.TrimSpace(before) {
		return model.TailoringChange{}, errNoTailoringChange
	}
	if after == "" || !utf8.ValidString(after) || utf8.RuneCountInString(after) > maxLen || strings.ContainsRune(after, '\x00') {
		return model.TailoringChange{}, model.ErrAIInvalidResponse
	}
	if kind == "summary" && suggestion.ExperienceID != "" {
		return model.TailoringChange{}, model.ErrAIInvalidResponse
	}
	evidence, err := validateTailoringEvidence(source, suggestion.Evidence, kind, entryID)
	if err != nil || !validTailoringReason(suggestion.Reason) {
		return model.TailoringChange{}, model.ErrAIInvalidResponse
	}
	// Metrics are especially easy for a model to fabricate. Every numeric token
	// must already occur in the cited source evidence, never merely in the Job.
	numbers := make(map[string]bool)
	for _, item := range evidence {
		for _, number := range tailoringNumbers.FindAllString(item.Quote, -1) {
			numbers[number] = true
		}
	}
	for _, number := range tailoringNumbers.FindAllString(after, -1) {
		if !numbers[number] {
			return model.TailoringChange{}, model.ErrAIInvalidResponse
		}
	}
	metricPairs := make(map[string]bool)
	for _, item := range evidence {
		for _, pair := range tailoringMetricPairs.FindAllString(item.Quote, -1) {
			metricPairs[normalizeTailoringMetric(pair)] = true
		}
	}
	for _, pair := range tailoringMetricPairs.FindAllString(after, -1) {
		if !metricPairs[normalizeTailoringMetric(pair)] {
			return model.TailoringChange{}, model.ErrAIInvalidResponse
		}
	}
	return model.TailoringChange{ID: uuid.NewString(), Kind: kind, Reason: strings.TrimSpace(suggestion.Reason), Evidence: evidence, Before: tailoringJSON(before), After: tailoringJSON(after)}, nil
}

func normalizeTailoringMetric(pair string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, pair)
}

func validTailoringReason(reason string) bool {
	return strings.TrimSpace(reason) != "" && utf8.ValidString(reason) && utf8.RuneCountInString(reason) <= 1000 && !strings.ContainsRune(reason, '\x00')
}

func validateTailoringEvidence(source *model.FullResumeDTO, items []ai.TailoringEvidence, kind, entryID string) ([]model.TailoringEvidence, error) {
	if len(items) == 0 || len(items) > 20 {
		return nil, model.ErrAIInvalidResponse
	}
	result := make([]model.TailoringEvidence, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Quote) == "" || utf8.RuneCountInString(item.Quote) > 10000 {
			return nil, model.ErrAIInvalidResponse
		}
		text := ""
		switch item.Section {
		case "summary":
			if item.EntryID == "" && source.Summary != nil {
				text = source.Summary.Content
			}
		case "experience":
			for _, exp := range source.Experiences {
				if exp.ID == item.EntryID {
					text = exp.Description
				}
			}
		case "skills":
			for _, skill := range source.Skills {
				if skill.ID == item.EntryID {
					text = skill.Name
				}
			}
		default:
			return nil, model.ErrAIInvalidResponse
		}
		if kind == "experience" && (item.Section != "experience" || item.EntryID != entryID) {
			return nil, model.ErrAIInvalidResponse
		}
		if kind == "skill_order" && item.Section != "skills" {
			return nil, model.ErrAIInvalidResponse
		}
		if !strings.Contains(text, item.Quote) {
			return nil, model.ErrAIInvalidResponse
		}
		result = append(result, model.TailoringEvidence{Section: item.Section, EntryID: item.EntryID, Quote: item.Quote})
	}
	return result, nil
}

func validateTailoringSkillOrder(source *model.FullResumeDTO, suggestion *ai.TailoringSkillSuggestion) (model.TailoringChange, error) {
	if len(suggestion.SkillIDs) != len(source.Skills) || len(source.Skills) < 2 || !validTailoringReason(suggestion.Reason) {
		return model.TailoringChange{}, model.ErrAIInvalidResponse
	}
	evidence, err := validateTailoringEvidence(source, suggestion.Evidence, "skill_order", "")
	if err != nil {
		return model.TailoringChange{}, err
	}
	byID := make(map[string]*model.SkillDTO, len(source.Skills))
	before := make([]model.TailoringSkillRef, 0, len(source.Skills))
	for _, skill := range source.Skills {
		byID[skill.ID] = skill
		before = append(before, model.TailoringSkillRef{ID: skill.ID, Name: skill.Name})
	}
	after := make([]model.TailoringSkillRef, 0, len(source.Skills))
	changed := false
	for i, id := range suggestion.SkillIDs {
		skill, ok := byID[id]
		if !ok {
			return model.TailoringChange{}, model.ErrAIInvalidResponse
		}
		delete(byID, id)
		after = append(after, model.TailoringSkillRef{ID: id, Name: skill.Name})
		if id != source.Skills[i].ID {
			changed = true
		}
	}
	if !changed {
		return model.TailoringChange{}, errNoTailoringChange
	}
	return model.TailoringChange{ID: uuid.NewString(), Kind: "skill_order", Reason: strings.TrimSpace(suggestion.Reason), Evidence: evidence, Before: tailoringJSON(before), After: tailoringJSON(after)}, nil
}

func tailoringJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("null")
	}
	return data
}

func tailoringVisible(source *model.FullResumeDTO, key string) bool {
	for _, section := range source.SectionOrder {
		if section.SectionKey == key {
			return section.IsVisible
		}
	}
	return false
}

func visibleTailoringSource(source *model.FullResumeDTO) *model.FullResumeDTO {
	visible := *source
	if !tailoringVisible(source, "summary") {
		visible.Summary = nil
	}
	if !tailoringVisible(source, "experience") {
		visible.Experiences = nil
	}
	if !tailoringVisible(source, "skills") {
		visible.Skills = nil
	}
	return &visible
}
