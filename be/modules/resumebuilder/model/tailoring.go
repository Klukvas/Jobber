package model

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrTailoringValidation      = errors.New("invalid tailoring selection")
	ErrTailoringNotFound        = errors.New("tailoring resource not found")
	ErrSourceChanged            = errors.New("source resume changed")
	ErrJobChanged               = errors.New("job changed")
	ErrProposalAlreadyApplied   = errors.New("proposal already applied with different selection or job link")
	ErrProposalExpired          = errors.New("proposal expired")
	ErrAIUnavailable            = errors.New("AI unavailable")
	ErrProposalStoreUnavailable = errors.New("proposal store unavailable")
	ErrAIInvalidResponse        = errors.New("invalid AI response")
	ErrAIRequestFailed          = errors.New("AI request failed")
	ErrAITimeout                = errors.New("AI request timed out")
)

type TailoringEvidence struct {
	Section string `json:"section"`
	EntryID string `json:"entry_id,omitempty"`
	Quote   string `json:"quote"`
}

type TailoringSkillRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TailoringChange keeps text and skill permutations typed at the JSON boundary.
type TailoringChange struct {
	ID           string              `json:"id"`
	Kind         string              `json:"kind"`
	Reason       string              `json:"reason"`
	Evidence     []TailoringEvidence `json:"evidence"`
	ExperienceID string              `json:"experience_id,omitempty"`
	Company      string              `json:"company"`
	Position     string              `json:"position"`
	Before       json.RawMessage     `json:"before"`
	After        json.RawMessage     `json:"after"`
}

type TailoringResource struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type TailoringProposalDTO struct {
	ProposalID   string            `json:"proposal_id"`
	ExpiresAt    time.Time         `json:"expires_at"`
	SourceResume TailoringResource `json:"source_resume"`
	Job          TailoringResource `json:"job"`
	Changes      []TailoringChange `json:"changes"`
}

// TailoringJob includes the exact saved context and link that must remain current.
type TailoringJob struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Company         string    `json:"company"`
	ResumeID        *string   `json:"resume_id"`
	ResumeBuilderID *string   `json:"resume_builder_id"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TailoringProposal struct {
	TailoringProposalDTO
	UserID              string         `json:"user_id"`
	Source              *FullResumeDTO `json:"source"`
	SourceHash          string         `json:"source_hash"`
	JobSnapshot         *TailoringJob  `json:"job_snapshot"`
	AcceptedFingerprint string         `json:"accepted_fingerprint,omitempty"`
}

type ApplyTailoringRequest struct {
	AcceptedChangeIDs []string `json:"accepted_change_ids" binding:"required,min=1,max=52,dive,required,max=100"`
	Title             string   `json:"title" binding:"required,max=255"`
}

type ApplyTailoringResult struct {
	ResumeBuilder *ResumeBuilderDTO `json:"resume_builder"`
	JobID         string            `json:"job_id"`
}
