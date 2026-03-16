package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// dbQuerier is satisfied by both *pgxpool.Pool and pgx.Tx.
type dbQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ResumeBuilderRepository implements ports.ResumeBuilderRepository.
type ResumeBuilderRepository struct {
	pool *pgxpool.Pool
	q    dbQuerier
}

// NewResumeBuilderRepository creates a new ResumeBuilderRepository.
func NewResumeBuilderRepository(pool *pgxpool.Pool) *ResumeBuilderRepository {
	return &ResumeBuilderRepository{pool: pool, q: pool}
}

// RunInTransaction executes fn within a database transaction.
// A temporary repository backed by the transaction is passed to fn.
func (r *ResumeBuilderRepository) RunInTransaction(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := &ResumeBuilderRepository{pool: r.pool, q: tx}
	if err := fn(txRepo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ResumeBuilderRepository) Create(ctx context.Context, rb *model.ResumeBuilder) error {
	rb.ID = uuid.New().String()
	now := time.Now().UTC()
	rb.CreatedAt = now
	rb.UpdatedAt = now

	query := `
		INSERT INTO resume_builders (id, user_id, title, template_id, font_family, primary_color, text_color, spacing, margin_top, margin_bottom, margin_left, margin_right, layout_mode, sidebar_width, font_size, skill_display, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.q.Exec(ctx, query,
		rb.ID, rb.UserID, rb.Title, rb.TemplateID, rb.FontFamily, rb.PrimaryColor, rb.TextColor,
		rb.Spacing, rb.MarginTop, rb.MarginBottom, rb.MarginLeft, rb.MarginRight,
		rb.LayoutMode, rb.SidebarWidth, rb.FontSize, rb.SkillDisplay,
		rb.CreatedAt, rb.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) GetByID(ctx context.Context, id string) (*model.ResumeBuilder, error) {
	query := `
		SELECT id, user_id, title, template_id, font_family, primary_color, text_color, spacing, margin_top, margin_bottom, margin_left, margin_right, layout_mode, sidebar_width, font_size, skill_display, created_at, updated_at
		FROM resume_builders WHERE id = $1
	`
	rb := &model.ResumeBuilder{}
	err := r.q.QueryRow(ctx, query, id).Scan(
		&rb.ID, &rb.UserID, &rb.Title, &rb.TemplateID, &rb.FontFamily, &rb.PrimaryColor, &rb.TextColor,
		&rb.Spacing, &rb.MarginTop, &rb.MarginBottom, &rb.MarginLeft, &rb.MarginRight,
		&rb.LayoutMode, &rb.SidebarWidth, &rb.FontSize, &rb.SkillDisplay,
		&rb.CreatedAt, &rb.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrResumeBuilderNotFound
		}
		return nil, err
	}
	return rb, nil
}

func (r *ResumeBuilderRepository) List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error) {
	query := `
		SELECT id, title, template_id, font_family, primary_color, text_color, spacing, margin_top, margin_bottom, margin_left, margin_right, layout_mode, sidebar_width, font_size, skill_display, created_at, updated_at
		FROM resume_builders WHERE user_id = $1 ORDER BY updated_at DESC
	`
	rows, err := r.q.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ResumeBuilderDTO
	for rows.Next() {
		dto := &model.ResumeBuilderDTO{}
		if err := rows.Scan(
			&dto.ID, &dto.Title, &dto.TemplateID, &dto.FontFamily, &dto.PrimaryColor, &dto.TextColor,
			&dto.Spacing, &dto.MarginTop, &dto.MarginBottom, &dto.MarginLeft, &dto.MarginRight,
			&dto.LayoutMode, &dto.SidebarWidth, &dto.FontSize, &dto.SkillDisplay,
			&dto.CreatedAt, &dto.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, dto)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) Update(ctx context.Context, rb *model.ResumeBuilder) error {
	query := `
		UPDATE resume_builders
		SET title = $1, template_id = $2, font_family = $3, primary_color = $4, text_color = $5,
		    spacing = $6, margin_top = $7, margin_bottom = $8, margin_left = $9, margin_right = $10,
		    layout_mode = $11, sidebar_width = $12, font_size = $13, skill_display = $14, updated_at = CURRENT_TIMESTAMP
		WHERE id = $15
		RETURNING updated_at
	`
	err := r.q.QueryRow(ctx, query,
		rb.Title, rb.TemplateID, rb.FontFamily, rb.PrimaryColor, rb.TextColor,
		rb.Spacing, rb.MarginTop, rb.MarginBottom, rb.MarginLeft, rb.MarginRight,
		rb.LayoutMode, rb.SidebarWidth, rb.FontSize, rb.SkillDisplay,
		rb.ID,
	).Scan(&rb.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrResumeBuilderNotFound
		}
		return err
	}
	return nil
}

func (r *ResumeBuilderRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_builders WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrResumeBuilderNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	var ownerID string
	err := r.q.QueryRow(ctx, `SELECT user_id FROM resume_builders WHERE id = $1`, resumeBuilderID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrResumeBuilderNotFound
		}
		return err
	}
	if ownerID != userID {
		return model.ErrNotOwner
	}
	return nil
}

func (r *ResumeBuilderRepository) GetFullResume(ctx context.Context, id string) (*model.FullResumeDTO, error) {
	rb, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load all sections in parallel using errgroup.
	// Each goroutine writes to its own local variable; results are assembled after Wait().
	g, gctx := errgroup.WithContext(ctx)

	var (
		contact      *model.ContactDTO
		summary      *model.SummaryDTO
		experiences  []*model.ExperienceDTO
		educations   []*model.EducationDTO
		skills       []*model.SkillDTO
		languages    []*model.LanguageDTO
		certs        []*model.CertificationDTO
		projects     []*model.ProjectDTO
		volunteering []*model.VolunteeringDTO
		customs      []*model.CustomSectionDTO
		sectionOrder []*model.SectionOrderDTO
	)

	g.Go(func() error {
		c, err := r.GetContact(gctx, id)
		if err != nil && !errors.Is(err, model.ErrSectionEntryNotFound) {
			return fmt.Errorf("load contact: %w", err)
		}
		if err == nil {
			contact = c.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		s, err := r.GetSummary(gctx, id)
		if err != nil && !errors.Is(err, model.ErrSectionEntryNotFound) {
			return fmt.Errorf("load summary: %w", err)
		}
		if err == nil {
			summary = &model.SummaryDTO{Content: s.Content}
		}
		return nil
	})

	g.Go(func() error {
		exps, err := r.ListExperiences(gctx, id)
		if err != nil {
			return fmt.Errorf("load experiences: %w", err)
		}
		experiences = make([]*model.ExperienceDTO, len(exps))
		for i, e := range exps {
			experiences[i] = e.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		edus, err := r.ListEducations(gctx, id)
		if err != nil {
			return fmt.Errorf("load educations: %w", err)
		}
		educations = make([]*model.EducationDTO, len(edus))
		for i, e := range edus {
			educations[i] = e.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		ss, err := r.ListSkills(gctx, id)
		if err != nil {
			return fmt.Errorf("load skills: %w", err)
		}
		skills = make([]*model.SkillDTO, len(ss))
		for i, s := range ss {
			skills[i] = s.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		ll, err := r.ListLanguages(gctx, id)
		if err != nil {
			return fmt.Errorf("load languages: %w", err)
		}
		languages = make([]*model.LanguageDTO, len(ll))
		for i, l := range ll {
			languages[i] = l.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		cc, err := r.ListCertifications(gctx, id)
		if err != nil {
			return fmt.Errorf("load certifications: %w", err)
		}
		certs = make([]*model.CertificationDTO, len(cc))
		for i, c := range cc {
			certs[i] = c.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		pp, err := r.ListProjects(gctx, id)
		if err != nil {
			return fmt.Errorf("load projects: %w", err)
		}
		projects = make([]*model.ProjectDTO, len(pp))
		for i, p := range pp {
			projects[i] = p.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		vv, err := r.ListVolunteering(gctx, id)
		if err != nil {
			return fmt.Errorf("load volunteering: %w", err)
		}
		volunteering = make([]*model.VolunteeringDTO, len(vv))
		for i, v := range vv {
			volunteering[i] = v.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		cc, err := r.ListCustomSections(gctx, id)
		if err != nil {
			return fmt.Errorf("load custom sections: %w", err)
		}
		customs = make([]*model.CustomSectionDTO, len(cc))
		for i, cs := range cc {
			customs[i] = cs.ToDTO()
		}
		return nil
	})

	g.Go(func() error {
		oo, err := r.ListSectionOrders(gctx, id)
		if err != nil {
			return fmt.Errorf("load section orders: %w", err)
		}
		sectionOrder = make([]*model.SectionOrderDTO, len(oo))
		for i, o := range oo {
			sectionOrder[i] = o.ToDTO()
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &model.FullResumeDTO{
		ResumeBuilderDTO: rb.ToDTO(),
		Contact:          contact,
		Summary:          summary,
		Experiences:      experiences,
		Educations:       educations,
		Skills:           skills,
		Languages:        languages,
		Certifications:   certs,
		Projects:         projects,
		Volunteering:     volunteering,
		CustomSections:   customs,
		SectionOrder:     sectionOrder,
	}, nil
}
