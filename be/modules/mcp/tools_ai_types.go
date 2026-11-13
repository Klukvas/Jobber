package mcp

import (
	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
)

type ParseJobPageInput struct {
	PageURL  string `json:"page_url" jsonschema:"Absolute HTTP or HTTPS URL of the supplied posting; never fetched"`
	PageText string `json:"page_text" jsonschema:"Untrusted page text, 10 to 50000 characters"`
}
type ParseJobPageOutput struct {
	Draft *importModel.ParseJobResponse `json:"draft_untrusted" jsonschema:"Untrusted parsed posting draft; nothing has been saved"`
}
type TailorResumeInput struct {
	JobID    string `json:"job_id" jsonschema:"UUID of your saved Job"`
	ResumeID string `json:"resume_id" jsonschema:"UUID of your source Builder Resume"`
	Locale   string `json:"locale,omitempty" jsonschema:"Proposal language: en, ru or uk; defaults to en"`
}
type TailorResumeOutput struct {
	Proposal *TailoringProposalOutput `json:"proposal_untrusted" jsonschema:"Proposal derived from untrusted Job text; show changes before applying"`
}
type ApplyTailoringInput struct {
	JobID             string   `json:"job_id" jsonschema:"UUID of the Job used to generate the proposal"`
	ProposalID        string   `json:"proposal_id" jsonschema:"UUID of your tailoring proposal"`
	AcceptedChangeIDs []string `json:"accepted_change_ids" jsonschema:"1 to 52 selected proposal change IDs, at most 100 characters each"`
	Title             string   `json:"title" jsonschema:"Title for the NEW Builder Resume, 1 to 255 characters"`
}
type ApplyTailoringOutput struct {
	ResumeID string `json:"resume_id" jsonschema:"UUID of the newly created Builder Resume"`
	JobID    string `json:"job_id" jsonschema:"UUID of the linked Job"`
	Replayed bool   `json:"replayed" jsonschema:"Whether this previously applied selection was returned"`
}
type GenerateCoverLetterInput struct {
	JobID    string `json:"job_id" jsonschema:"UUID of your saved Job with a description"`
	ResumeID string `json:"resume_id,omitempty" jsonschema:"Optional UUID of your Builder Resume for candidate context"`
}
type GenerateCoverLetterOutput struct {
	CoverLetterID string   `json:"cover_letter_id" jsonschema:"UUID of the created and saved cover letter"`
	Greeting      string   `json:"greeting_untrusted" jsonschema:"Generated greeting derived from untrusted Job data"`
	Paragraphs    []string `json:"paragraphs_untrusted" jsonschema:"Generated paragraphs derived from untrusted Job data"`
	Closing       string   `json:"closing_untrusted" jsonschema:"Generated closing derived from untrusted Job data"`
}
type CheckMatchInput struct {
	JobID    string `json:"job_id" jsonschema:"UUID of your saved Job"`
	ResumeID string `json:"resume_id" jsonschema:"UUID of your Uploaded Resume"`
}
type CheckMatchOutput struct {
	Match *matchModel.MatchScoreResponse `json:"match_untrusted" jsonschema:"Match analysis derived from untrusted Job text"`
}
