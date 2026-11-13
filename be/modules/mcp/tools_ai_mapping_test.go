package mcp

import (
	"context"
	"encoding/json"
	"testing"

	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/require"
)

type projectionTailoring struct {
	Tailoring
	proposal *rbModel.TailoringProposalDTO
}

func (p projectionTailoring) Generate(context.Context, string, string, string, string) (*rbModel.TailoringProposalDTO, error) {
	return p.proposal, nil
}

func TestTailoringProposalSDKOutputAcceptsTextAndSkillLists(t *testing.T) {
	cases := []struct {
		name     string
		raw      json.RawMessage
		expected any
	}{
		{"text change", json.RawMessage(`"Source text"`), "Source text"},
		{"skill order", json.RawMessage(`[{"id":"skill","name":"Go"}]`), []any{map[string]any{"id": "skill", "name": "Go"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := append(json.RawMessage(nil), tc.raw...)
			proposal := &rbModel.TailoringProposalDTO{ProposalID: proposalA, Changes: []rbModel.TailoringChange{{ID: tailoredA, Before: tc.raw, After: tc.raw}}}
			fake := newFakeAITools()
			deps := fake.deps()
			deps.Tailoring = projectionTailoring{proposal: proposal}
			jobs := newFakeJobs()
			client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(deps)), userA)
			result := decodeOutput[TailorResumeOutput](t, callTool(t, client, "tailor_resume", TailorResumeInput{JobID: jobA, ResumeID: builderA}))
			require.Equal(t, tc.expected, result.Proposal.Changes[0].Before)
			require.Equal(t, tc.expected, result.Proposal.Changes[0].After)
			require.Equal(t, original, proposal.Changes[0].Before)
			require.Equal(t, original, proposal.Changes[0].After)
		})
	}
}
func TestMapTailoringProposalRejectsInvalidJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		proposal *rbModel.TailoringProposalDTO
	}{
		{"nil result", nil},
		{"invalid before", &rbModel.TailoringProposalDTO{Changes: []rbModel.TailoringChange{{Before: json.RawMessage("private invalid JSON")}}}},
		{"invalid after", &rbModel.TailoringProposalDTO{Changes: []rbModel.TailoringChange{{After: json.RawMessage("private invalid JSON")}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mapTailoringProposal(tc.proposal)
			require.Nil(t, result)
			require.ErrorIs(t, err, errAIProvider)
			require.NotContains(t, err.Error(), "private")
		})
	}
}
