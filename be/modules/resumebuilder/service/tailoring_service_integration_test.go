//go:build integration

package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type integrationTailoringLimits struct {
	pool    *pgxpool.Pool
	maximum int
}

func (l *integrationTailoringLimits) CheckLimit(context.Context, string, string) error { return nil }
func (l *integrationTailoringLimits) RecordAIUsage(context.Context, string) error      { return nil }
func (l *integrationTailoringLimits) ResourceLimit(ctx context.Context, user, resource string) (int, error) {
	var value int
	err := l.pool.QueryRow(ctx, `SELECT $1::int`, l.maximum).Scan(&value)
	return value, err
}

type tailoringIntegrationFixture struct {
	pool                    *pgxpool.Pool
	repo                    *repository.ResumeBuilderRepository
	userID, jobID, sourceID string
	limits                  *integrationTailoringLimits
	service                 *TailoringService
	store                   *tailoringTestStore
	source                  *model.FullResumeDTO
}

func newTailoringIntegrationFixture(t *testing.T, maxConns int32) *tailoringIntegrationFixture {
	t.Helper()
	dsn := os.Getenv("TAILORING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TAILORING_TEST_DATABASE_URL must point to the isolated synthetic database")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "jobber_builder_verify", cfg.ConnConfig.Database, "refusing writes outside the isolated verification database")
	require.Contains(t, []string{"127.0.0.1", "localhost"}, cfg.ConnConfig.Host)
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	user := uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Tailoring integration','test-only-unused')`, user, user+"@tailoring.example.test")
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, user)
		assert.NoError(t, err)
		pool.Close()
	})
	repo := repository.NewResumeBuilderRepository(pool)
	limits := &integrationTailoringLimits{pool: pool, maximum: 100}
	builder := NewResumeBuilderService(repo, limits)
	source, err := builder.Create(t.Context(), user, &model.CreateResumeBuilderRequest{Title: "Source", TemplateID: DefaultTemplateID})
	require.NoError(t, err)
	require.NoError(t, repo.UpsertSummary(t.Context(), &model.Summary{ResumeBuilderID: source.ID, Content: "Built reliable Go services."}))
	require.NoError(t, repo.UpsertContact(t.Context(), &model.Contact{ResumeBuilderID: source.ID, FullName: "Synthetic Example", Email: "synthetic@example.test"}))
	require.NoError(t, repo.CreateExperience(t.Context(), &model.Experience{ResumeBuilderID: source.ID, Company: "Example", Position: "Engineer", Description: "Built Go services.", SortOrder: 0}))
	require.NoError(t, repo.CreateEducation(t.Context(), &model.Education{ResumeBuilderID: source.ID, Institution: "Example University", Degree: "BSc", GPA: "A"}))
	require.NoError(t, repo.CreateSkill(t.Context(), &model.Skill{ResumeBuilderID: source.ID, Name: "Go", Level: "expert", SortOrder: 0}))
	require.NoError(t, repo.CreateSkill(t.Context(), &model.Skill{ResumeBuilderID: source.ID, Name: "SQL", Level: "advanced", SortOrder: 1}))
	require.NoError(t, repo.CreateLanguage(t.Context(), &model.Language{ResumeBuilderID: source.ID, Name: "English", Proficiency: "fluent"}))
	require.NoError(t, repo.CreateCertification(t.Context(), &model.Certification{ResumeBuilderID: source.ID, Name: "Certification", Issuer: "Example"}))
	require.NoError(t, repo.CreateProject(t.Context(), &model.Project{ResumeBuilderID: source.ID, Name: "Project", Description: "Private unchanged project"}))
	require.NoError(t, repo.CreateVolunteering(t.Context(), &model.Volunteering{ResumeBuilderID: source.ID, Organization: "Community", Role: "Helper"}))
	require.NoError(t, repo.CreateCustomSection(t.Context(), &model.CustomSection{ResumeBuilderID: source.ID, Title: "Custom", Content: "Unchanged custom content"}))
	stage := uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO stage_templates(id,user_id,name,"order") VALUES($1,$2,'Saved',0)`, stage, user)
	require.NoError(t, err)
	require.NoError(t, repo.UpsertSectionOrder(t.Context(), source.ID, []*model.SectionOrder{{SectionKey: "experience", SortOrder: 2, Column: "main", IsVisible: false}}))
	job := uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id) VALUES($1,$2,'Go Engineer','Build reliable Go services',$3)`, job, user, stage)
	require.NoError(t, err)
	full, err := repo.GetFullResume(t.Context(), user, source.ID)
	require.NoError(t, err)
	store := &tailoringTestStore{}
	client := &tailoringTestAI{result: &ai.ResumeTailoringSuggestions{Summary: &ai.TailoringTextSuggestion{After: "Engineer building reliable Go services.", Reason: "Emphasize relevant engineering", Evidence: []ai.TailoringEvidence{{Section: "summary", Quote: full.Summary.Content}}}}}
	return &tailoringIntegrationFixture{pool: pool, repo: repo, userID: user, jobID: job, sourceID: source.ID, limits: limits, service: NewTailoringService(repo, repo, client, store, limits), store: store, source: full}
}
func (f *tailoringIntegrationFixture) generate(t *testing.T) (*model.TailoringProposalDTO, *model.ApplyTailoringRequest) {
	t.Helper()
	p, err := f.service.Generate(t.Context(), f.userID, f.jobID, f.sourceID, "en")
	require.NoError(t, err)
	return p, &model.ApplyTailoringRequest{Title: "Tailored copy", AcceptedChangeIDs: []string{p.Changes[0].ID}}
}

func TestTailoringPostgresPoolOneAndFullCopy(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 1)
	p, req := f.generate(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	result, replayed, err := f.service.Apply(ctx, f.userID, f.jobID, p.ProposalID, req)
	require.NoError(t, err, "Apply must not acquire a second pool connection while holding its transaction")
	require.False(t, replayed)
	copy, err := f.repo.GetFullResume(t.Context(), f.userID, result.ResumeBuilder.ID)
	require.NoError(t, err)
	after, err := f.repo.GetFullResume(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	assert.Equal(t, f.source, after)
	assert.Equal(t, "Engineer building reliable Go services.", copy.Summary.Content)
	assert.Equal(t, after.Contact, copy.Contact)
	assert.Equal(t, after.SectionOrder, copy.SectionOrder)
	assert.Equal(t, after.TemplateID, copy.TemplateID)
	assert.Equal(t, after.SkillDisplay, copy.SkillDisplay)
	assert.Len(t, copy.Experiences, 1)
	assert.Len(t, copy.Educations, 1)
	assert.Len(t, copy.Skills, 2)
	assert.Len(t, copy.Languages, 1)
	assert.Len(t, copy.Certifications, 1)
	assert.Len(t, copy.Projects, 1)
	assert.Len(t, copy.Volunteering, 1)
	assert.Len(t, copy.CustomSections, 1)
	assert.NotEqual(t, after.Experiences[0].ID, copy.Experiences[0].ID)
	assert.Equal(t, after.Experiences[0].Description, copy.Experiences[0].Description)
	assert.Equal(t, after.Projects[0].Description, copy.Projects[0].Description)
	assert.Equal(t, after.CustomSections[0].Content, copy.CustomSections[0].Content)
	expected, err := applyTailoringChanges(after, f.store.proposal.Changes)
	require.NoError(t, err)
	assert.Equal(t, normalizedTailoringCopy(t, expected), normalizedTailoringCopy(t, copy))
	again, replayed, err := f.service.Apply(ctx, f.userID, f.jobID, p.ProposalID, req)
	require.NoError(t, err)
	assert.True(t, replayed)
	assert.Equal(t, result.ResumeBuilder.ID, again.ResumeBuilder.ID)
}

func TestTailoringPostgresConcurrentLastSlot(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 5)
	p, req := f.generate(t)
	f.limits.maximum = 2
	builder := NewResumeBuilderService(f.repo, f.limits)
	importer := NewImportServiceWithDeps(f.repo, &MockResumeTextParser{ParseResumeTextFunc: func(context.Context, string) (*ai.ParsedResume, error) {
		return &ai.ParsedResume{FullName: "Imported synthetic"}, nil
	}}, f.limits, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 4)
	var wg sync.WaitGroup
	for _, action := range []func() error{func() error { _, _, err := f.service.Apply(ctx, f.userID, f.jobID, p.ProposalID, req); return err }, func() error {
		_, err := builder.Create(ctx, f.userID, &model.CreateResumeBuilderRequest{Title: "Concurrent create"})
		return err
	}, func() error { _, err := builder.Duplicate(ctx, f.userID, f.sourceID); return err }, func() error {
		_, err := importer.ImportFromText(ctx, f.userID, "synthetic input", "Concurrent import")
		return err
	}} {
		wg.Go(func() { <-start; results <- action() })
	}
	close(start)
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, subModel.ErrLimitReached) {
			limited++
		} else {
			t.Errorf("unexpected concurrent error: %v", err)
		}
	}
	assert.Equal(t, 1, success)
	assert.Equal(t, 3, limited)
	var count int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM resume_builders WHERE user_id=$1`, f.userID).Scan(&count))
	assert.Equal(t, 2, count)
}

func TestTailoringPostgresStaleAndRollback(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 2)
	p, req := f.generate(t)
	exp, err := f.repo.GetExperienceByID(t.Context(), f.sourceID, f.source.Experiences[0].ID)
	require.NoError(t, err)
	exp.Description = "Edited after proposal"
	require.NoError(t, f.repo.UpdateExperience(t.Context(), exp))
	_, _, err = f.service.Apply(t.Context(), f.userID, f.jobID, p.ProposalID, req)
	require.ErrorIs(t, err, model.ErrSourceChanged)
	_, err = f.repo.GetByID(t.Context(), f.userID, p.ProposalID)
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	rollbackID := uuid.NewString()
	err = f.repo.RunTailoringTransaction(t.Context(), f.userID, func(tx ports.TailoringTransaction) error {
		require.NoError(t, tx.CreateWithID(t.Context(), tailoredResumeMetadata(f.userID, rollbackID, "Rollback", f.source)))
		return tx.LinkTailoredResume(t.Context(), f.userID, uuid.NewString(), rollbackID)
	})
	require.ErrorIs(t, err, model.ErrTailoringNotFound)
	_, err = f.repo.GetByID(t.Context(), f.userID, rollbackID)
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
}

func normalizedTailoringCopy(t *testing.T, full *model.FullResumeDTO) map[string]any {
	t.Helper()
	data, err := json.Marshal(full)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(data, &value))
	for _, key := range []string{"id", "title", "created_at", "updated_at"} {
		delete(value, key)
	}
	for _, section := range []string{"experiences", "educations", "skills", "languages", "certifications", "projects", "volunteering", "custom_sections"} {
		if items, ok := value[section].([]any); ok {
			for _, item := range items {
				delete(item.(map[string]any), "id")
			}
		}
	}
	return value
}
