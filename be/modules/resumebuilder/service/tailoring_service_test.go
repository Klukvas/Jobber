package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tailoringTestAI struct {
	result *ai.ResumeTailoringSuggestions
	err    error
	calls  int
	facts  string
}

func (c *tailoringTestAI) TailorResume(ctx context.Context, job, facts, locale string) (*ai.ResumeTailoringSuggestions, error) {
	c.calls++
	c.facts = facts
	return c.result, c.err
}

type tailoringTestStore struct {
	proposal *model.TailoringProposal
	err      error
}

func (s *tailoringTestStore) Save(ctx context.Context, p *model.TailoringProposal) error {
	if s.err != nil {
		return s.err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	var copy model.TailoringProposal
	if err := json.Unmarshal(data, &copy); err != nil {
		return err
	}
	s.proposal = &copy
	return nil
}
func (s *tailoringTestStore) Get(ctx context.Context, user, id string) (*model.TailoringProposal, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.proposal == nil || s.proposal.UserID != user || s.proposal.ProposalID != id || time.Now().After(s.proposal.ExpiresAt) {
		return nil, model.ErrProposalExpired
	}
	data, err := json.Marshal(s.proposal)
	if err != nil {
		return nil, err
	}
	var copy model.TailoringProposal
	err = json.Unmarshal(data, &copy)
	return &copy, err
}

type tailoringTestRepo struct {
	*MockResumeBuilderRepository
	mu         sync.Mutex
	source     *model.FullResumeDTO
	job        *model.TailoringJob
	copies     map[string]*model.ResumeBuilder
	creates    int
	linkErr    error
	fullCopies map[string]*model.FullResumeDTO
	failSource bool
}

func (r *tailoringTestRepo) GetTailoringJob(ctx context.Context, user, id string) (*model.TailoringJob, error) {
	if user != "user" || id != r.job.ID {
		return nil, model.ErrTailoringNotFound
	}
	copy := *r.job
	return &copy, nil
}
func (r *tailoringTestRepo) RunTailoringTransaction(ctx context.Context, user string, fn func(ports.TailoringTransaction) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := *r.job
	oldCopies := make(map[string]*model.ResumeBuilder)
	for id, copy := range r.copies {
		oldCopies[id] = copy
	}
	err := fn(r)
	if err != nil {
		r.job = &old
		r.copies = oldCopies
	}
	return err
}
func (r *tailoringTestRepo) LockTailoringSource(ctx context.Context, user, id string) (*model.FullResumeDTO, error) {
	if r.failSource || user != "user" || id != r.source.ID {
		return nil, model.ErrTailoringNotFound
	}
	return r.source, nil
}
func (r *tailoringTestRepo) CheckResumeBuilderCapacity(context.Context, string, int) error {
	return nil
}

func (r *tailoringTestRepo) CreateWithID(ctx context.Context, resume *model.ResumeBuilder) error {
	r.creates++
	copy := *resume
	r.copies[resume.ID] = &copy
	return nil
}
func (r *tailoringTestRepo) LinkTailoredResume(ctx context.Context, user, job, id string) error {
	if r.linkErr != nil {
		return r.linkErr
	}
	r.job.ResumeID = nil
	r.job.ResumeBuilderID = &id
	r.job.UpdatedAt = time.Now()
	return nil
}

func tailoringFixture(t *testing.T) (*TailoringService, *tailoringTestRepo, *tailoringTestStore, *tailoringTestAI, *MockLimitChecker) {
	t.Helper()
	source := newFullResumeForAI()
	source.SectionOrder = []*model.SectionOrderDTO{{SectionKey: "summary", IsVisible: true}, {SectionKey: "experience", IsVisible: true}, {SectionKey: "skills", IsVisible: true}}
	source.TemplateID = DefaultTemplateID
	source.FontFamily = "Georgia"
	source.MarginTop = 38
	source.LayoutMode = "double-left"
	source.Skills[0].Level = "expert"
	job := &model.TailoringJob{ID: "job", Title: "Go developer", Description: "Build reliable Go services", UpdatedAt: time.Now()}
	repo := &tailoringTestRepo{MockResumeBuilderRepository: &MockResumeBuilderRepository{}, source: source, job: job, copies: map[string]*model.ResumeBuilder{}, fullCopies: map[string]*model.FullResumeDTO{}}
	repo.GetFullResumeFunc = func(ctx context.Context, id string) (*model.FullResumeDTO, error) {
		if id != source.ID {
			return nil, model.ErrResumeBuilderNotFound
		}
		return source, nil
	}
	repo.GetByIDFunc = func(ctx context.Context, id string) (*model.ResumeBuilder, error) {
		copy, ok := repo.copies[id]
		if !ok {
			return nil, model.ErrResumeBuilderNotFound
		}
		return copy, nil
	}
	store := &tailoringTestStore{}
	client := &tailoringTestAI{result: &ai.ResumeTailoringSuggestions{Summary: &ai.TailoringTextSuggestion{After: "Software engineer with 10 years of experience.", Reason: "Emphasize engineering", Evidence: []ai.TailoringEvidence{{Section: "summary", Quote: source.Summary.Content}}}}}
	limits := &MockLimitChecker{}
	return NewTailoringService(repo, repo, client, store, limits), repo, store, client, limits
}

func TestTailoringServiceGenerate(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*TailoringService, *tailoringTestRepo, *tailoringTestStore, *tailoringTestAI, *MockLimitChecker)
		want  error
		calls int
	}{
		{name: "success", calls: 1},
		{name: "foreign job", setup: func(s *TailoringService, r *tailoringTestRepo, _ *tailoringTestStore, _ *tailoringTestAI, _ *MockLimitChecker) {
			r.job.ID = "foreign"
		}, want: model.ErrTailoringNotFound},
		{name: "foreign source", setup: func(s *TailoringService, r *tailoringTestRepo, _ *tailoringTestStore, _ *tailoringTestAI, _ *MockLimitChecker) {
			r.GetFullResumeFunc = func(context.Context, string) (*model.FullResumeDTO, error) {
				return nil, model.ErrResumeBuilderNotFound
			}
		}, want: model.ErrTailoringNotFound},
		{name: "unavailable AI", setup: func(s *TailoringService, _ *tailoringTestRepo, _ *tailoringTestStore, _ *tailoringTestAI, _ *MockLimitChecker) {
			s.client = nil
		}, want: model.ErrAIUnavailable},
		{name: "AI quota", setup: func(_ *TailoringService, _ *tailoringTestRepo, _ *tailoringTestStore, _ *tailoringTestAI, l *MockLimitChecker) {
			l.CheckLimitFunc = func(context.Context, string, string) error { return subModel.ErrLimitReached }
		}, want: subModel.ErrLimitReached},
		{name: "provider error", setup: func(_ *TailoringService, _ *tailoringTestRepo, _ *tailoringTestStore, c *tailoringTestAI, _ *MockLimitChecker) {
			c.err = errors.New("provider unavailable")
		}, want: model.ErrAIRequestFailed, calls: 1},
		{name: "provider timeout", setup: func(_ *TailoringService, _ *tailoringTestRepo, _ *tailoringTestStore, c *tailoringTestAI, _ *MockLimitChecker) {
			c.err = context.DeadlineExceeded
		}, want: model.ErrAITimeout, calls: 1},
		{name: "invalid provider JSON", setup: func(_ *TailoringService, _ *tailoringTestRepo, _ *tailoringTestStore, c *tailoringTestAI, _ *MockLimitChecker) {
			c.err = ai.ErrInvalidTailoringResponse
		}, want: model.ErrAIInvalidResponse, calls: 1},
		{name: "store unavailable", setup: func(_ *TailoringService, _ *tailoringTestRepo, s *tailoringTestStore, _ *tailoringTestAI, _ *MockLimitChecker) {
			s.err = model.ErrProposalStoreUnavailable
		}, want: model.ErrProposalStoreUnavailable, calls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, r, store, c, l := tailoringFixture(t)
			if test.setup != nil {
				test.setup(s, r, store, c, l)
			}
			result, err := s.Generate(t.Context(), "user", "job", r.source.ID, "en")
			if test.want != nil {
				require.ErrorIs(t, err, test.want)
			} else {
				require.NoError(t, err)
				require.Len(t, result.Changes, 1)
				assert.NotEmpty(t, result.ProposalID)
				assert.Equal(t, 0, r.creates)
			}
			assert.Equal(t, test.calls, c.calls)
		})
	}
}

func TestTailoringServiceApply(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*tailoringTestRepo, *tailoringTestStore, *MockLimitChecker, *model.ApplyTailoringRequest)
		want   error
	}{
		{name: "success"},
		{name: "stale section", change: func(r *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			r.source.Experiences[0].Description = "Changed while reviewing"
		}, want: model.ErrSourceChanged},
		{name: "stale job", change: func(r *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			r.job.Title = "Changed Job"
		}, want: model.ErrJobChanged},
		{name: "invalid selection", change: func(_ *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, req *model.ApplyTailoringRequest) {
			req.AcceptedChangeIDs = []string{"unknown"}
		}, want: model.ErrTailoringValidation},
		{name: "duplicate selection", change: func(_ *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, req *model.ApplyTailoringRequest) {
			req.AcceptedChangeIDs = append(req.AcceptedChangeIDs, req.AcceptedChangeIDs[0])
		}, want: model.ErrTailoringValidation},
		{name: "resume quota", change: func(_ *tailoringTestRepo, _ *tailoringTestStore, l *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			l.CheckLimitFunc = func(context.Context, string, string) error { return subModel.ErrLimitReached }
		}, want: subModel.ErrLimitReached},
		{name: "link failure rolls back copy", change: func(r *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			r.linkErr = model.ErrJobChanged
		}, want: model.ErrJobChanged},
		{name: "expired", change: func(_ *tailoringTestRepo, store *tailoringTestStore, _ *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			store.proposal.ExpiresAt = time.Now().Add(-time.Second)
		}, want: model.ErrProposalExpired},
		{name: "source deleted", change: func(r *tailoringTestRepo, _ *tailoringTestStore, _ *MockLimitChecker, _ *model.ApplyTailoringRequest) {
			r.failSource = true
		}, want: model.ErrTailoringNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, r, store, _, limits := tailoringFixture(t)
			proposal, err := s.Generate(t.Context(), "user", "job", r.source.ID, "en")
			require.NoError(t, err)
			req := &model.ApplyTailoringRequest{Title: "  For Go Job  ", AcceptedChangeIDs: []string{proposal.Changes[0].ID}}
			if test.change != nil {
				test.change(r, store, limits, req)
			}
			sourceBefore := hashTailoring(r.source)
			result, replay, err := s.Apply(t.Context(), "user", "job", proposal.ProposalID, req)
			if test.want != nil {
				require.ErrorIs(t, err, test.want)
				assert.Empty(t, r.copies)
				assert.Nil(t, r.job.ResumeBuilderID)
			} else {
				require.NoError(t, err)
				assert.False(t, replay)
				assert.Equal(t, "For Go Job", result.ResumeBuilder.Title)
				assert.Equal(t, proposal.ProposalID, result.ResumeBuilder.ID)
				assert.Equal(t, r.source.MarginTop, result.ResumeBuilder.MarginTop)
				assert.Equal(t, r.source.LayoutMode, result.ResumeBuilder.LayoutMode)
				assert.Equal(t, proposal.ProposalID, *r.job.ResumeBuilderID)
			}
			assert.Equal(t, sourceBefore, hashTailoring(r.source))
		})
	}
}

func TestTailoringServiceApplyReplay(t *testing.T) {
	s, r, store, _, limits := tailoringFixture(t)
	proposal, err := s.Generate(t.Context(), "user", "job", r.source.ID, "en")
	require.NoError(t, err)
	req := &model.ApplyTailoringRequest{Title: "Job copy", AcceptedChangeIDs: []string{proposal.Changes[0].ID}}
	first, replayed, err := s.Apply(t.Context(), "user", "job", proposal.ProposalID, req)
	require.NoError(t, err)
	require.False(t, replayed)
	r.source.Summary.Content = "Changed after Apply"
	limits.CheckLimitFunc = func(context.Context, string, string) error { return subModel.ErrLimitReached }
	second, replayed, err := s.Apply(t.Context(), "user", "job", proposal.ProposalID, req)
	require.NoError(t, err)
	assert.True(t, replayed)
	assert.Equal(t, first.ResumeBuilder.ID, second.ResumeBuilder.ID)
	assert.Equal(t, 1, r.creates)
	changed := *req
	changed.Title = "Another title"
	_, _, err = s.Apply(t.Context(), "user", "job", proposal.ProposalID, &changed)
	assert.ErrorIs(t, err, model.ErrProposalAlreadyApplied)
	r.job.ResumeBuilderID = nil
	_, _, err = s.Apply(t.Context(), "user", "job", proposal.ProposalID, req)
	assert.ErrorIs(t, err, model.ErrProposalAlreadyApplied)
	assert.Nil(t, r.job.ResumeBuilderID)
	assert.NotEmpty(t, store.proposal.AcceptedFingerprint)
}

func TestTailoringServiceConcurrentApply(t *testing.T) {
	s, r, _, _, _ := tailoringFixture(t)
	proposal, err := s.Generate(t.Context(), "user", "job", r.source.ID, "en")
	require.NoError(t, err)
	req := &model.ApplyTailoringRequest{Title: "Job copy", AcceptedChangeIDs: []string{proposal.Changes[0].ID}}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Go(func() { _, _, err := s.Apply(t.Context(), "user", "job", proposal.ProposalID, req); results <- err })
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	assert.Equal(t, 1, r.creates)
}
