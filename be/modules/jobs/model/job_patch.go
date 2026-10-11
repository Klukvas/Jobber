package model

// JobPatch is the set of card fields one PATCH actually wants to change,
// normalized for the repository. A nil field means "not in the request — leave
// the column alone"; a non-nil field is written.
//
// The write is deliberately field-scoped rather than a whole-row update: two
// PATCHes that touch different fields must not overwrite each other, and
// neither may touch the pipeline-owned columns (current_stage_template_id,
// current_stage_id, applied_at) that Move/AddStage maintain under a row lock.
type JobPatch struct {
	// CompanyID, ResumeID and ResumeBuilderID hold a UUID to link, or the
	// empty string to clear the link (written as SQL NULL).
	CompanyID       *string
	ResumeID        *string
	ResumeBuilderID *string

	Title       *string
	Source      *string
	URL         *string
	Description *string
	IsArchived  *bool
}
