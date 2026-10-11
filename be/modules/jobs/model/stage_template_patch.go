package model

// StageTemplatePatch is the set of pipeline-column fields one PATCH actually
// wants to change, normalized for the repository. A nil field means "not in the
// request — leave the column alone"; a non-nil field is written.
//
// Field-scoped like JobPatch, and for the same reason: the label, the position
// and the meaning of a column are three separate things, so a rename and a
// reclassification that overlap must both survive instead of one whole-row
// write overwriting the other's field.
type StageTemplatePatch struct {
	Name  *string
	Order *int
	Kind  *StageKind
}
