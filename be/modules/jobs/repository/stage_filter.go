package repository

import "fmt"

// stageFilter restricts Jobs to their current pipeline Stage. UUID values are
// validated by callers and always bound, just like the company filter.
func stageFilter(stageTemplateID string, args *[]any) string {
	if stageTemplateID == "" {
		return ""
	}
	*args = append(*args, stageTemplateID)
	return fmt.Sprintf(" AND j.current_stage_template_id = $%d::uuid", len(*args))
}
