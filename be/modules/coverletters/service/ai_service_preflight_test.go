package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type preflightLimitChecker struct {
	CheckLimitFunc    func(context.Context, string, string) error
	RecordAIUsageFunc func(context.Context, string) error
}

func (m *preflightLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	return m.CheckLimitFunc(ctx, userID, resource)
}
func (m *preflightLimitChecker) RecordAIUsage(ctx context.Context, userID string) error {
	return m.RecordAIUsageFunc(ctx, userID)
}

func validAISourceRepos(userID, coverLetterID string) (*MockCoverLetterRepository, *mockAIResumeBuilderRepo) {
	resumeID := "rb-1"
	return &MockCoverLetterRepository{GetByIDFunc: func(context.Context, string) (*model.CoverLetter, error) {
			return &model.CoverLetter{ID: coverLetterID, UserID: userID, ResumeBuilderID: &resumeID}, nil
		}}, &mockAIResumeBuilderRepo{GetFullResumeFunc: func(context.Context, string) (*rbModel.FullResumeDTO, error) {
			return &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{ID: resumeID}, Experiences: []*rbModel.ExperienceDTO{{Description: "Built APIs."}}}, nil
		}}
}

func TestAIService_GeneratePreflight(t *testing.T) {
	for _, tc := range []struct {
		name         string
		job          string
		link         *string
		ownershipErr error
		loadErr      error
		resume       *rbModel.FullResumeDTO
		wantErr      error
	}{
		{name: "empty job", wantErr: model.ErrAIJobDescriptionRequired},
		{name: "whitespace job", job: " \t\n\u00a0 ", wantErr: model.ErrAIJobDescriptionRequired},
		{name: "missing link", job: "Engineer", wantErr: model.ErrAICandidateSourceRequired},
		{name: "blank link", job: "Engineer", link: stringPointer("  "), wantErr: model.ErrAICandidateSourceRequired},
		{name: "not owned", job: "Engineer", link: stringPointer("rb-1"), ownershipErr: rbModel.ErrNotOwner, wantErr: model.ErrAICandidateSourceRequired},
		{name: "missing resume", job: "Engineer", link: stringPointer("rb-1"), loadErr: rbModel.ErrResumeBuilderNotFound, wantErr: model.ErrAICandidateSourceRequired},
		{name: "nil resume", job: "Engineer", link: stringPointer("rb-1"), wantErr: model.ErrAICandidateSourceRequired},
		{name: "empty resume", job: "Engineer", link: stringPointer("rb-1"), resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}}, wantErr: model.ErrAICandidateSourceRequired},
		{name: "contact only", job: "Engineer", link: stringPointer("rb-1"), resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}, Contact: &rbModel.ContactDTO{FullName: "Alice"}}, wantErr: model.ErrAICandidateSourceRequired},
		{name: "blank experience", job: "Engineer", link: stringPointer("rb-1"), resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}, Experiences: []*rbModel.ExperienceDTO{{Position: "Developer", Company: "Acme", Description: " \n "}}}, wantErr: model.ErrAICandidateSourceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks, calls, usage, loads := 0, 0, 0, 0
			clRepo := &MockCoverLetterRepository{GetByIDFunc: func(context.Context, string) (*model.CoverLetter, error) {
				return &model.CoverLetter{UserID: "user-1", ResumeBuilderID: tc.link}, nil
			}}
			rbRepo := &mockAIResumeBuilderRepo{
				VerifyOwnershipFunc: func(context.Context, string, string) error { return tc.ownershipErr },
				GetFullResumeFunc:   func(context.Context, string) (*rbModel.FullResumeDTO, error) { loads++; return tc.resume, tc.loadErr },
			}
			limiter := &preflightLimitChecker{
				CheckLimitFunc:    func(context.Context, string, string) error { checks++; return nil },
				RecordAIUsageFunc: func(context.Context, string) error { usage++; return nil },
			}
			client := &mockAICoverLetterAIClient{GenerateCoverLetterFunc: func(context.Context, string, string, string, string, string) (*ai.CoverLetterContent, error) {
				calls++
				return &ai.CoverLetterContent{}, nil
			}}
			result, err := NewAIService(clRepo, rbRepo, client, limiter).Generate(context.Background(), "user-1", "cl-1", tc.job)
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, result)
			assert.Zero(t, checks, "preflight must precede allowance/reservation")
			assert.Zero(t, calls)
			assert.Zero(t, usage)
			if tc.ownershipErr != nil {
				assert.Zero(t, loads, "do not read another user's resume")
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func TestAIService_GenerateSourceAndUsage(t *testing.T) {
	for _, tc := range []struct {
		name        string
		resume      *rbModel.FullResumeDTO
		providerErr error
	}{
		{name: "experience without role or company", resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}, Experiences: []*rbModel.ExperienceDTO{{Description: "Built APIs."}}}},
		{name: "summary credential fallback", resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}, Summary: &rbModel.SummaryDTO{Content: "Studying computer science."}}},
		{name: "skill credential fallback", resume: &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{}, Skills: []*rbModel.SkillDTO{{Name: "Go"}}}},
		{name: "invalid response has no usage", providerErr: ai.ErrInvalidCoverLetterResponse},
		{name: "provider failure has no usage", providerErr: errors.New("provider unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clRepo, rbRepo := validAISourceRepos("user-1", "cl-1")
			if tc.resume != nil {
				rbRepo.GetFullResumeFunc = func(context.Context, string) (*rbModel.FullResumeDTO, error) { return tc.resume, nil }
			}
			checks, calls, usage := 0, 0, 0
			limiter := &preflightLimitChecker{
				CheckLimitFunc:    func(context.Context, string, string) error { checks++; return nil },
				RecordAIUsageFunc: func(context.Context, string) error { usage++; return nil },
			}
			want := &ai.CoverLetterContent{Greeting: "Dear Hiring Manager,", Paragraphs: []string{"I am interested in the role.", "Built APIs.", "I welcome a discussion."}, Closing: "Sincerely,"}
			client := &mockAICoverLetterAIClient{GenerateCoverLetterFunc: func(_ context.Context, _, _, _, job, source string) (*ai.CoverLetterContent, error) {
				calls++
				assert.Equal(t, "Engineer", job)
				assert.True(t, ai.HasCoverLetterCandidateSource(source))
				if tc.providerErr != nil {
					return nil, tc.providerErr
				}
				return want, nil
			}}
			result, err := NewAIService(clRepo, rbRepo, client, limiter).Generate(context.Background(), "user-1", "cl-1", "Engineer")
			assert.Equal(t, 1, checks)
			assert.Equal(t, 1, calls)
			if tc.providerErr != nil {
				require.ErrorIs(t, err, tc.providerErr)
				assert.Nil(t, result)
				assert.Zero(t, usage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, want, result)
				assert.Equal(t, 1, usage)
			}
		})
	}
}
