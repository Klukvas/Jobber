package mcp

import (
	"time"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

type ListResumesInput struct{}

type ResumeItem struct {
	Kind      string    `json:"kind" jsonschema:"Resume kind: builder is editable; uploaded is a read-only reference"`
	ID        string    `json:"id" jsonschema:"Resume UUID"`
	Title     string    `json:"title" jsonschema:"Resume title"`
	UpdatedAt time.Time `json:"updated_at" jsonschema:"Last update timestamp in RFC3339 format"`
}

type ListResumesOutput struct {
	Resumes []ResumeItem `json:"resumes" jsonschema:"All Builder Resumes and Uploaded Resume references; no uploaded content or URLs"`
}

type GetResumeInput struct {
	ResumeID string `json:"resume_id" jsonschema:"UUID of a Builder Resume owned by the caller; Uploaded Resumes are not supported"`
}

type GetResumeOutput struct {
	Resume *rb.FullResumeDTO `json:"resume" jsonschema:"Full structured Builder Resume including section entry IDs for updates"`
}

type UpdateResumeSectionInput struct {
	ResumeID string         `json:"resume_id" jsonschema:"UUID of the caller's Builder Resume"`
	Section  string         `json:"section" jsonschema:"Section to edit: contact, summary, experience, education, skill, language, certification, project, volunteering, custom"`
	Op       string         `json:"op" jsonschema:"Use upsert for contact/summary; create or update for list sections"`
	EntryID  string         `json:"entry_id,omitempty" jsonschema:"Entry UUID from get_resume; required only for update"`
	Data     map[string]any `json:"data" jsonschema:"Section fields only; no IDs or sort_order. Contact: full_name/email/phone/location/website/linkedin/github. Summary: content. Experience: company/position/location/start_date/end_date/is_current/description. Education: institution/degree/field_of_study/start_date/end_date/is_current/gpa/description. Skill: name/level. Language: name/proficiency. Certification: name/issuer/issue_date/expiry_date/url. Project: name/url/start_date/end_date/description. Volunteering: organization/role/start_date/end_date/description. Custom: title/content. Strings use existing Resume Builder limits; is_current is boolean. Update changes only supplied fields; upsert replaces the whole contact or summary. Text allows tabs and newlines; other control characters are rejected."`
}

type UpdateResumeSectionOutput struct {
	Entry        map[string]any `json:"entry" jsonschema:"Updated section entry; includes its UUID for list sections"`
	SnapshotNote string         `json:"snapshot_note" jsonschema:"How to undo this edit using resume snapshots"`
}

type ResumeSnapshotItem struct {
	ID        string    `json:"id" jsonschema:"Snapshot UUID for restore_resume_snapshot"`
	Source    string    `json:"source" jsonschema:"Snapshot origin: mcp or restore"`
	CreatedAt time.Time `json:"created_at" jsonschema:"Snapshot creation time in RFC3339 format"`
}

type ListResumeSnapshotsOutput struct {
	Snapshots []ResumeSnapshotItem `json:"snapshots" jsonschema:"Saved versions of this Builder Resume without their private content"`
}

type RestoreResumeSnapshotInput struct {
	SnapshotID string `json:"snapshot_id" jsonschema:"UUID from list_resume_snapshots; must belong to the caller"`
}

type RestoreResumeSnapshotOutput struct {
	Resume       ResumeItem `json:"resume" jsonschema:"Restored Builder Resume summary"`
	SnapshotNote string     `json:"snapshot_note" jsonschema:"The restore creates an undo snapshot of the previous content"`
}
