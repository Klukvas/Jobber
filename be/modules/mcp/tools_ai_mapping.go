package mcp

import (
	"encoding/json"
	"time"

	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// RawMessage infers as a byte array in SDK schemas, although it marshals to the
// embedded JSON value. Project it to JSON values so real text edits validate.
type TailoringChangeOutput struct {
	ID           string                      `json:"id" jsonschema:"Proposal change UUID"`
	Kind         string                      `json:"kind" jsonschema:"Change kind: summary, experience or skill_order"`
	Reason       string                      `json:"reason" jsonschema:"Untrusted explanation of Job relevance"`
	Evidence     []rbModel.TailoringEvidence `json:"evidence" jsonschema:"Candidate source evidence for this change"`
	ExperienceID string                      `json:"experience_id,omitempty" jsonschema:"Experience UUID when editing an experience"`
	Company      string                      `json:"company" jsonschema:"Candidate employer associated with this change"`
	Position     string                      `json:"position" jsonschema:"Candidate position associated with this change"`
	Before       any                         `json:"before" jsonschema:"Original text or ordered skill list"`
	After        any                         `json:"after" jsonschema:"Proposed text or ordered skill list"`
}
type TailoringProposalOutput struct {
	ProposalID   string                    `json:"proposal_id" jsonschema:"Proposal UUID for apply_tailoring"`
	ExpiresAt    time.Time                 `json:"expires_at" jsonschema:"Proposal expiration timestamp"`
	SourceResume rbModel.TailoringResource `json:"source_resume" jsonschema:"Source Builder Resume preserved by tailoring"`
	Job          rbModel.TailoringResource `json:"job" jsonschema:"Job associated with the proposal"`
	Changes      []TailoringChangeOutput   `json:"changes" jsonschema:"Untrusted proposed changes; select accepted change IDs before applying"`
}

func mapTailoringProposal(proposal *rbModel.TailoringProposalDTO) (*TailoringProposalOutput, error) {
	if proposal == nil {
		return nil, errAIProvider
	}
	changes := make([]TailoringChangeOutput, 0, len(proposal.Changes))
	for _, change := range proposal.Changes {
		var before, after any
		if len(change.Before) > 0 {
			if err := json.Unmarshal(change.Before, &before); err != nil {
				return nil, errAIProvider
			}
		}
		if len(change.After) > 0 {
			if err := json.Unmarshal(change.After, &after); err != nil {
				return nil, errAIProvider
			}
		}
		changes = append(changes, TailoringChangeOutput{ID: change.ID, Kind: change.Kind, Reason: change.Reason, Evidence: change.Evidence, ExperienceID: change.ExperienceID, Company: change.Company, Position: change.Position, Before: before, After: after})
	}
	return &TailoringProposalOutput{ProposalID: proposal.ProposalID, ExpiresAt: proposal.ExpiresAt, SourceResume: proposal.SourceResume, Job: proposal.Job, Changes: changes}, nil
}
