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
)

// dbQuerier is satisfied by *pgxpool.Pool, *pgxpool.Conn and pgx.Tx.
type dbQuerier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ResumeBuilderRepository implements ports.ResumeBuilderRepository.
type ResumeBuilderRepository struct {
	pool PgxPool
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
	// Acquire a single connection for the duration of this call.
	// The previous implementation launched ~11 concurrent goroutines via errgroup,
	// each acquiring its own pool connection. On the resume editor (re-renders per
	// keystroke) this exhausted the pool (typically 10-25 conns) and caused
	// cascading timeouts. Running all section queries sequentially on one connection
	// keeps pool pressure at exactly 1 connection per call.
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// connRepo routes all queries through the single acquired connection.
	connRepo := &ResumeBuilderRepository{pool: r.pool, q: conn}

	rb, err := connRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	c, err := connRepo.GetContact(ctx, id)
	var contact *model.ContactDTO
	if err != nil && !errors.Is(err, model.ErrSectionEntryNotFound) {
		return nil, fmt.Errorf("load contact: %w", err)
	}
	if err == nil {
		contact = c.ToDTO()
	}

	s, err := connRepo.GetSummary(ctx, id)
	var summary *model.SummaryDTO
	if err != nil && !errors.Is(err, model.ErrSectionEntryNotFound) {
		return nil, fmt.Errorf("load summary: %w", err)
	}
	if err == nil {
		summary = &model.SummaryDTO{Content: s.Content}
	}

	exps, err := connRepo.ListExperiences(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load experiences: %w", err)
	}
	experiences := make([]*model.ExperienceDTO, len(exps))
	for i, e := range exps {
		experiences[i] = e.ToDTO()
	}

	edus, err := connRepo.ListEducations(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load educations: %w", err)
	}
	educations := make([]*model.EducationDTO, len(edus))
	for i, e := range edus {
		educations[i] = e.ToDTO()
	}

	ss, err := connRepo.ListSkills(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	skills := make([]*model.SkillDTO, len(ss))
	for i, sk := range ss {
		skills[i] = sk.ToDTO()
	}

	ll, err := connRepo.ListLanguages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load languages: %w", err)
	}
	languages := make([]*model.LanguageDTO, len(ll))
	for i, l := range ll {
		languages[i] = l.ToDTO()
	}

	cc, err := connRepo.ListCertifications(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load certifications: %w", err)
	}
	certs := make([]*model.CertificationDTO, len(cc))
	for i, cert := range cc {
		certs[i] = cert.ToDTO()
	}

	pp, err := connRepo.ListProjects(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	projects := make([]*model.ProjectDTO, len(pp))
	for i, p := range pp {
		projects[i] = p.ToDTO()
	}

	vv, err := connRepo.ListVolunteering(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load volunteering: %w", err)
	}
	volunteering := make([]*model.VolunteeringDTO, len(vv))
	for i, v := range vv {
		volunteering[i] = v.ToDTO()
	}

	cs, err := connRepo.ListCustomSections(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load custom sections: %w", err)
	}
	customs := make([]*model.CustomSectionDTO, len(cs))
	for i, cust := range cs {
		customs[i] = cust.ToDTO()
	}

	oo, err := connRepo.ListSectionOrders(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load section orders: %w", err)
	}
	sectionOrder := make([]*model.SectionOrderDTO, len(oo))
	for i, o := range oo {
		sectionOrder[i] = o.ToDTO()
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
