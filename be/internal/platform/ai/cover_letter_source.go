package ai

// HasCoverLetterCandidateSource shares generation's complete source catalog and
// limits, including credential anchors when no work description is available.
// Plain text and empty structured resumes have no eligible typed source unit.
func HasCoverLetterCandidateSource(resumeJSON string) bool {
	catalog, err := resumeGroundingCatalog(resumeJSON, false)
	return err == nil && validateGroundingInput(catalog, 60000) == nil && len(letterAnchorFacts(catalog)) > 0
}
