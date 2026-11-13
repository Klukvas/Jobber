package mcp

import (
	"testing"

	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestCreateJobPreflightDoesNotPersistCompany(t *testing.T) {
	tests := []struct {
		name, stage string
		limitErr    error
		want        string
	}{
		{name: "foreign Stage", stage: stageA, want: "Stage not found"},
		{name: "missing Stage", stage: "88888888-8888-4888-8888-888888888888", want: "Stage not found"},
		{name: "Jobs limit", limitErr: subModel.ErrLimitReached, want: "Upgrade"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jobs := newFakeJobs()
			jobs.createError = test.limitErr
			companies := &fakeCompanies{items: make(map[string][]*companyModel.CompanyDTO)}
			client := toolClientForServer(t, NewServer(jobs, companies, jobs, nil), userB)
			result := callTool(t, client, "create_job", CreateJobInput{Title: "Role", Company: "New company", StageID: test.stage})
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].(*sdk.TextContent).Text, test.want)
			require.Empty(t, companies.items)
			require.Len(t, jobs.jobs, 1)
		})
	}
}
