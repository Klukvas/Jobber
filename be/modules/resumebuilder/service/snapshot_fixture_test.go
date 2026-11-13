package service

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type snapshotTestRepo struct {
	*MockResumeBuilderRepository
	mu        sync.Mutex
	t         *testing.T
	current   *model.FullResumeDTO
	snapshots []*model.ResumeSnapshot
	writeErr  error
	pruneErr  error
	writes    int
}

func newSnapshotTestRepo(t *testing.T) *snapshotTestRepo {
	t.Helper()
	r := &snapshotTestRepo{t: t, current: snapshotTestResume(), MockResumeBuilderRepository: &MockResumeBuilderRepository{}}
	r.UpdateFunc = func(_ context.Context, rb *model.ResumeBuilder) error {
		r.writes++
		if r.writeErr != nil {
			return r.writeErr
		}
		metadata := rb.ToDTO()
		metadata.CreatedAt = r.current.CreatedAt
		r.current.ResumeBuilderDTO = metadata
		return nil
	}
	r.bindSnapshotSectionWrites()
	return r
}

func (r *snapshotTestRepo) RunSnapshotTransaction(ctx context.Context, fn func(ports.SnapshotTransaction) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	before := snapshotTestClone(r.t, r.current)
	snapshots := snapshotTestClone(r.t, r.snapshots)
	if err := fn(r); err != nil {
		r.current, r.snapshots = before, snapshots
		return err
	}
	return nil
}

func (r *snapshotTestRepo) LockResume(_ context.Context, user, resume string) error {
	if user != "owner" || resume != r.current.ID {
		return model.ErrResumeBuilderNotFound
	}
	return nil
}

func (r *snapshotTestRepo) GetFullResume(ctx context.Context, user, resume string) (*model.FullResumeDTO, error) {
	if err := r.LockResume(ctx, user, resume); err != nil {
		return nil, err
	}
	return snapshotTestClone(r.t, r.current), nil
}

func (r *snapshotTestRepo) GetSnapshot(_ context.Context, user, id string) (*model.ResumeSnapshot, error) {
	for _, snapshot := range r.snapshots {
		if snapshot.ID == id && snapshot.UserID == user {
			return snapshotTestClone(r.t, snapshot), nil
		}
	}
	return nil, model.ErrResumeBuilderNotFound
}

func (r *snapshotTestRepo) HasRecentSnapshot(_ context.Context, resume, source string, since time.Time) (bool, error) {
	for _, snapshot := range r.snapshots {
		if snapshot.ResumeBuilderID == resume && snapshot.Source == source && snapshot.CreatedAt.After(since) {
			return true, nil
		}
	}
	return false, nil
}

func (r *snapshotTestRepo) InsertSnapshot(_ context.Context, snapshot *model.ResumeSnapshot) error {
	r.writes++
	r.snapshots = append(r.snapshots, snapshotTestClone(r.t, snapshot))
	return nil
}

func (r *snapshotTestRepo) PruneSnapshots(_ context.Context, resume string, keep int) error {
	r.writes++
	if r.pruneErr != nil {
		return r.pruneErr
	}
	sort.Slice(r.snapshots, func(i, j int) bool { return r.snapshots[i].CreatedAt.After(r.snapshots[j].CreatedAt) })
	count := 0
	retained := make([]*model.ResumeSnapshot, 0)
	for _, snapshot := range r.snapshots {
		if snapshot.ResumeBuilderID == resume {
			count++
			if count > keep {
				continue
			}
		}
		retained = append(retained, snapshot)
	}
	r.snapshots = retained
	return nil
}

func (r *snapshotTestRepo) ListSnapshots(ctx context.Context, user, resume string) ([]*model.ResumeSnapshotDTO, error) {
	if err := r.LockResume(ctx, user, resume); err != nil {
		return nil, err
	}
	items := make([]*model.ResumeSnapshotDTO, 0)
	for _, snapshot := range r.snapshots {
		if snapshot.UserID == user && snapshot.ResumeBuilderID == resume {
			items = append(items, &model.ResumeSnapshotDTO{ID: snapshot.ID, Source: snapshot.Source, CreatedAt: snapshot.CreatedAt})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *snapshotTestRepo) ClearResumeSections(_ context.Context, _ string) error {
	r.writes++
	r.current = &model.FullResumeDTO{ResumeBuilderDTO: r.current.ResumeBuilderDTO}
	return nil
}

func (r *snapshotTestRepo) bindSnapshotSectionWrites() {
	r.UpsertContactFunc = func(_ context.Context, v *model.Contact) error { r.current.Contact = v.ToDTO(); return nil }
	r.UpsertSummaryFunc = func(_ context.Context, v *model.Summary) error {
		r.current.Summary = &model.SummaryDTO{Content: v.Content}
		return nil
	}
	r.CreateExperienceFunc = func(_ context.Context, v *model.Experience) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Experiences = append(r.current.Experiences, copy.ToDTO())
		return nil
	}
	r.CreateEducationFunc = func(_ context.Context, v *model.Education) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Educations = append(r.current.Educations, copy.ToDTO())
		return nil
	}
	r.CreateSkillFunc = func(_ context.Context, v *model.Skill) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Skills = append(r.current.Skills, copy.ToDTO())
		return nil
	}
	r.CreateLanguageFunc = func(_ context.Context, v *model.Language) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Languages = append(r.current.Languages, copy.ToDTO())
		return nil
	}
	r.bindSnapshotAdditionalSections()
}

func (r *snapshotTestRepo) bindSnapshotAdditionalSections() {
	r.CreateCertificationFunc = func(_ context.Context, v *model.Certification) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Certifications = append(r.current.Certifications, copy.ToDTO())
		return nil
	}
	r.CreateProjectFunc = func(_ context.Context, v *model.Project) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Projects = append(r.current.Projects, copy.ToDTO())
		return nil
	}
	r.CreateVolunteeringFunc = func(_ context.Context, v *model.Volunteering) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.Volunteering = append(r.current.Volunteering, copy.ToDTO())
		return nil
	}
	r.CreateCustomSectionFunc = func(_ context.Context, v *model.CustomSection) error {
		copy := *v
		copy.ID = uuid.NewString()
		r.current.CustomSections = append(r.current.CustomSections, copy.ToDTO())
		return nil
	}
	r.UpsertSectionOrderFunc = func(_ context.Context, _ string, orders []*model.SectionOrder) error {
		for _, order := range orders {
			r.current.SectionOrder = append(r.current.SectionOrder, order.ToDTO())
		}
		return nil
	}
}

func snapshotTestClone[T any](t *testing.T, value T) T {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	var copy T
	require.NoError(t, json.Unmarshal(data, &copy))
	return copy
}

func snapshotTestResume() *model.FullResumeDTO {
	return &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "resume", Title: "Original", TemplateID: DefaultTemplateID,
			FontFamily: "Arial", PrimaryColor: "#112233", TextColor: "#223344", Spacing: 110,
			MarginTop: 20, MarginBottom: 30, MarginLeft: 40, MarginRight: 50,
			LayoutMode: "double-left", SidebarWidth: 30, FontSize: 14, SkillDisplay: "dots"},
		Contact: &model.ContactDTO{FullName: "Synthetic", Email: "test@example.test", Phone: "123", Location: "City",
			Website: "https://example.test", LinkedIn: "linkedin", GitHub: "github"},
		Summary: &model.SummaryDTO{Content: "Original summary"},
		Experiences: []*model.ExperienceDTO{{ID: "exp", Company: "Company", Position: "Engineer", Location: "Remote",
			StartDate: "2020", EndDate: "2025", IsCurrent: true, Description: "Experience", SortOrder: 2}},
		Educations: []*model.EducationDTO{{ID: "edu", Institution: "University", Degree: "BSc", FieldOfStudy: "CS",
			StartDate: "2015", EndDate: "2019", IsCurrent: true, GPA: "A", Description: "Education", SortOrder: 1}},
		Skills:         []*model.SkillDTO{{ID: "skill", Name: "Go", Level: "expert", SortOrder: 3}},
		Languages:      []*model.LanguageDTO{{ID: "lang", Name: "English", Proficiency: "fluent", SortOrder: 4}},
		Certifications: []*model.CertificationDTO{{ID: "cert", Name: "Cert", Issuer: "Issuer", IssueDate: "2020", ExpiryDate: "2030", URL: "https://example.test/cert", SortOrder: 5}},
		Projects:       []*model.ProjectDTO{{ID: "project", Name: "Project", URL: "https://example.test/project", StartDate: "2021", EndDate: "2022", Description: "Project description", SortOrder: 6}},
		Volunteering:   []*model.VolunteeringDTO{{ID: "volunteer", Organization: "Community", Role: "Helper", StartDate: "2023", EndDate: "2024", Description: "Volunteer", SortOrder: 7}},
		CustomSections: []*model.CustomSectionDTO{{ID: "custom", Title: "Custom", Content: "Custom content", SortOrder: 8}},
		SectionOrder:   []*model.SectionOrderDTO{{SectionKey: "skills", SortOrder: 1, IsVisible: false, Column: "sidebar"}},
	}
}
