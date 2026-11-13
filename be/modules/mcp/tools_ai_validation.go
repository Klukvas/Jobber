package mcp

import (
	"net/url"
	"unicode/utf8"
)

func validatePage(input ParseJobPageInput) error {
	if err := validateText(input.PageText, "page_text", 50000, true); err != nil {
		return err
	}
	if utf8.RuneCountInString(input.PageText) < 10 {
		return validationError("page_text must contain at least 10 characters.")
	}
	if err := validateText(input.PageURL, "page_url", 2048, true); err != nil {
		return err
	}
	parsed, err := url.Parse(input.PageURL)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return validationError("page_url must be an absolute HTTP or HTTPS URL without credentials.")
	}
	return nil
}
func canonicalPair(job, resume string) (string, string, error) {
	jobID, err := canonicalID(job, "job_id")
	if err != nil {
		return "", "", err
	}
	resumeID, err := canonicalID(resume, "resume_id")
	return jobID, resumeID, err
}
func validateApply(input ApplyTailoringInput) (ApplyTailoringInput, error) {
	job, err := canonicalID(input.JobID, "job_id")
	if err != nil {
		return input, err
	}
	proposal, err := canonicalID(input.ProposalID, "proposal_id")
	if err != nil {
		return input, err
	}
	if err := validateText(input.Title, "title", 255, true); err != nil {
		return input, err
	}
	if len(input.AcceptedChangeIDs) < 1 || len(input.AcceptedChangeIDs) > 52 {
		return input, validationError("Select 1 to 52 accepted_change_ids.")
	}
	seen := make(map[string]bool)
	for _, id := range input.AcceptedChangeIDs {
		if err := validateText(id, "accepted_change_ids", 100, true); err != nil {
			return input, err
		}
		if seen[id] {
			return input, validationError("accepted_change_ids must be unique.")
		}
		seen[id] = true
	}
	return ApplyTailoringInput{JobID: job, ProposalID: proposal, Title: input.Title, AcceptedChangeIDs: append([]string(nil), input.AcceptedChangeIDs...)}, nil
}
