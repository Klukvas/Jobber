package repository

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Contact (1:1) ---

func (r *ResumeBuilderRepository) UpsertContact(ctx context.Context, c *model.Contact) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO resume_contacts (id, resume_builder_id, full_name, email, phone, location, website, linkedin, github, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (resume_builder_id)
		DO UPDATE SET full_name = $3, email = $4, phone = $5, location = $6, website = $7, linkedin = $8, github = $9, updated_at = $11
	`
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	_, err := r.q.Exec(ctx, query,
		c.ID, c.ResumeBuilderID, c.FullName, c.Email, c.Phone, c.Location, c.Website, c.LinkedIn, c.GitHub, now, now,
	)
	return err
}

func (r *ResumeBuilderRepository) GetContact(ctx context.Context, resumeBuilderID string) (*model.Contact, error) {
	query := `SELECT id, resume_builder_id, full_name, email, phone, location, website, linkedin, github, created_at, updated_at FROM resume_contacts WHERE resume_builder_id = $1`
	c := &model.Contact{}
	err := r.q.QueryRow(ctx, query, resumeBuilderID).Scan(
		&c.ID, &c.ResumeBuilderID, &c.FullName, &c.Email, &c.Phone, &c.Location, &c.Website, &c.LinkedIn, &c.GitHub, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return c, nil
}

// --- Summary (1:1) ---

func (r *ResumeBuilderRepository) UpsertSummary(ctx context.Context, s *model.Summary) error {
	now := time.Now().UTC()
	query := `
		INSERT INTO resume_summaries (id, resume_builder_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (resume_builder_id)
		DO UPDATE SET content = $3, updated_at = $5
	`
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	_, err := r.q.Exec(ctx, query, s.ID, s.ResumeBuilderID, s.Content, now, now)
	return err
}

func (r *ResumeBuilderRepository) GetSummary(ctx context.Context, resumeBuilderID string) (*model.Summary, error) {
	query := `SELECT id, resume_builder_id, content, created_at, updated_at FROM resume_summaries WHERE resume_builder_id = $1`
	s := &model.Summary{}
	err := r.q.QueryRow(ctx, query, resumeBuilderID).Scan(&s.ID, &s.ResumeBuilderID, &s.Content, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return s, nil
}

// --- Experiences ---

func (r *ResumeBuilderRepository) CreateExperience(ctx context.Context, exp *model.Experience) error {
	exp.ID = uuid.New().String()
	now := time.Now().UTC()
	exp.CreatedAt = now
	exp.UpdatedAt = now

	query := `
		INSERT INTO resume_experiences (id, resume_builder_id, company, position, location, start_date, end_date, is_current, description, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.q.Exec(ctx, query,
		exp.ID, exp.ResumeBuilderID, exp.Company, exp.Position, exp.Location,
		exp.StartDate, exp.EndDate, exp.IsCurrent, exp.Description, exp.SortOrder,
		exp.CreatedAt, exp.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) UpdateExperience(ctx context.Context, exp *model.Experience) error {
	query := `
		UPDATE resume_experiences
		SET company = $1, position = $2, location = $3, start_date = $4, end_date = $5, is_current = $6, description = $7, sort_order = $8
		WHERE id = $9 AND resume_builder_id = $10
	`
	tag, err := r.q.Exec(ctx, query,
		exp.Company, exp.Position, exp.Location, exp.StartDate, exp.EndDate,
		exp.IsCurrent, exp.Description, exp.SortOrder, exp.ID, exp.ResumeBuilderID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_experiences WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error) {
	query := `
		SELECT id, resume_builder_id, company, position, location, start_date, end_date, is_current, description, sort_order, created_at, updated_at
		FROM resume_experiences WHERE resume_builder_id = $1 ORDER BY sort_order
	`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Experience
	for rows.Next() {
		e := &model.Experience{}
		if err := rows.Scan(
			&e.ID, &e.ResumeBuilderID, &e.Company, &e.Position, &e.Location,
			&e.StartDate, &e.EndDate, &e.IsCurrent, &e.Description, &e.SortOrder,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error) {
	query := `SELECT id, resume_builder_id, company, position, location, start_date, end_date, is_current, description, sort_order, created_at, updated_at
		FROM resume_experiences WHERE id = $1 AND resume_builder_id = $2`
	e := &model.Experience{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&e.ID, &e.ResumeBuilderID, &e.Company, &e.Position, &e.Location,
		&e.StartDate, &e.EndDate, &e.IsCurrent, &e.Description, &e.SortOrder,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return e, nil
}

// --- Educations ---

func (r *ResumeBuilderRepository) CreateEducation(ctx context.Context, edu *model.Education) error {
	edu.ID = uuid.New().String()
	now := time.Now().UTC()
	edu.CreatedAt = now
	edu.UpdatedAt = now

	query := `
		INSERT INTO resume_educations (id, resume_builder_id, institution, degree, field_of_study, start_date, end_date, is_current, gpa, description, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.q.Exec(ctx, query,
		edu.ID, edu.ResumeBuilderID, edu.Institution, edu.Degree, edu.FieldOfStudy,
		edu.StartDate, edu.EndDate, edu.IsCurrent, edu.GPA, edu.Description, edu.SortOrder,
		edu.CreatedAt, edu.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) UpdateEducation(ctx context.Context, edu *model.Education) error {
	query := `
		UPDATE resume_educations
		SET institution = $1, degree = $2, field_of_study = $3, start_date = $4, end_date = $5, is_current = $6, gpa = $7, description = $8, sort_order = $9
		WHERE id = $10 AND resume_builder_id = $11
	`
	tag, err := r.q.Exec(ctx, query,
		edu.Institution, edu.Degree, edu.FieldOfStudy, edu.StartDate, edu.EndDate,
		edu.IsCurrent, edu.GPA, edu.Description, edu.SortOrder, edu.ID, edu.ResumeBuilderID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_educations WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListEducations(ctx context.Context, resumeBuilderID string) ([]*model.Education, error) {
	query := `
		SELECT id, resume_builder_id, institution, degree, field_of_study, start_date, end_date, is_current, gpa, description, sort_order, created_at, updated_at
		FROM resume_educations WHERE resume_builder_id = $1 ORDER BY sort_order
	`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Education
	for rows.Next() {
		e := &model.Education{}
		if err := rows.Scan(
			&e.ID, &e.ResumeBuilderID, &e.Institution, &e.Degree, &e.FieldOfStudy,
			&e.StartDate, &e.EndDate, &e.IsCurrent, &e.GPA, &e.Description, &e.SortOrder,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*model.Education, error) {
	query := `SELECT id, resume_builder_id, institution, degree, field_of_study, start_date, end_date, is_current, gpa, description, sort_order, created_at, updated_at
		FROM resume_educations WHERE id = $1 AND resume_builder_id = $2`
	e := &model.Education{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&e.ID, &e.ResumeBuilderID, &e.Institution, &e.Degree, &e.FieldOfStudy,
		&e.StartDate, &e.EndDate, &e.IsCurrent, &e.GPA, &e.Description, &e.SortOrder,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return e, nil
}

// --- Skills ---

func (r *ResumeBuilderRepository) CreateSkill(ctx context.Context, skill *model.Skill) error {
	skill.ID = uuid.New().String()
	now := time.Now().UTC()
	skill.CreatedAt = now
	skill.UpdatedAt = now

	query := `INSERT INTO resume_skills (id, resume_builder_id, name, level, sort_order, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.q.Exec(ctx, query, skill.ID, skill.ResumeBuilderID, skill.Name, skill.Level, skill.SortOrder, skill.CreatedAt, skill.UpdatedAt)
	return err
}

func (r *ResumeBuilderRepository) UpdateSkill(ctx context.Context, skill *model.Skill) error {
	query := `UPDATE resume_skills SET name = $1, level = $2, sort_order = $3 WHERE id = $4 AND resume_builder_id = $5`
	tag, err := r.q.Exec(ctx, query, skill.Name, skill.Level, skill.SortOrder, skill.ID, skill.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_skills WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListSkills(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error) {
	query := `SELECT id, resume_builder_id, name, level, sort_order, created_at, updated_at FROM resume_skills WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Skill
	for rows.Next() {
		s := &model.Skill{}
		if err := rows.Scan(&s.ID, &s.ResumeBuilderID, &s.Name, &s.Level, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error) {
	query := `SELECT id, resume_builder_id, name, level, sort_order, created_at, updated_at FROM resume_skills WHERE id = $1 AND resume_builder_id = $2`
	s := &model.Skill{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&s.ID, &s.ResumeBuilderID, &s.Name, &s.Level, &s.SortOrder, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return s, nil
}

// --- Languages ---

func (r *ResumeBuilderRepository) CreateLanguage(ctx context.Context, lang *model.Language) error {
	lang.ID = uuid.New().String()
	now := time.Now().UTC()
	lang.CreatedAt = now
	lang.UpdatedAt = now

	query := `INSERT INTO resume_languages (id, resume_builder_id, name, proficiency, sort_order, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.q.Exec(ctx, query, lang.ID, lang.ResumeBuilderID, lang.Name, lang.Proficiency, lang.SortOrder, lang.CreatedAt, lang.UpdatedAt)
	return err
}

func (r *ResumeBuilderRepository) UpdateLanguage(ctx context.Context, lang *model.Language) error {
	query := `UPDATE resume_languages SET name = $1, proficiency = $2, sort_order = $3 WHERE id = $4 AND resume_builder_id = $5`
	tag, err := r.q.Exec(ctx, query, lang.Name, lang.Proficiency, lang.SortOrder, lang.ID, lang.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_languages WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*model.Language, error) {
	query := `SELECT id, resume_builder_id, name, proficiency, sort_order, created_at, updated_at FROM resume_languages WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Language
	for rows.Next() {
		l := &model.Language{}
		if err := rows.Scan(&l.ID, &l.ResumeBuilderID, &l.Name, &l.Proficiency, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*model.Language, error) {
	query := `SELECT id, resume_builder_id, name, proficiency, sort_order, created_at, updated_at FROM resume_languages WHERE id = $1 AND resume_builder_id = $2`
	l := &model.Language{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&l.ID, &l.ResumeBuilderID, &l.Name, &l.Proficiency, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return l, nil
}

// --- Certifications ---

func (r *ResumeBuilderRepository) CreateCertification(ctx context.Context, cert *model.Certification) error {
	cert.ID = uuid.New().String()
	now := time.Now().UTC()
	cert.CreatedAt = now
	cert.UpdatedAt = now

	query := `INSERT INTO resume_certifications (id, resume_builder_id, name, issuer, issue_date, expiry_date, url, sort_order, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.q.Exec(ctx, query,
		cert.ID, cert.ResumeBuilderID, cert.Name, cert.Issuer, cert.IssueDate, cert.ExpiryDate, cert.URL, cert.SortOrder,
		cert.CreatedAt, cert.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) UpdateCertification(ctx context.Context, cert *model.Certification) error {
	query := `UPDATE resume_certifications SET name = $1, issuer = $2, issue_date = $3, expiry_date = $4, url = $5, sort_order = $6 WHERE id = $7 AND resume_builder_id = $8`
	tag, err := r.q.Exec(ctx, query, cert.Name, cert.Issuer, cert.IssueDate, cert.ExpiryDate, cert.URL, cert.SortOrder, cert.ID, cert.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_certifications WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error) {
	query := `SELECT id, resume_builder_id, name, issuer, issue_date, expiry_date, url, sort_order, created_at, updated_at FROM resume_certifications WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Certification
	for rows.Next() {
		c := &model.Certification{}
		if err := rows.Scan(&c.ID, &c.ResumeBuilderID, &c.Name, &c.Issuer, &c.IssueDate, &c.ExpiryDate, &c.URL, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error) {
	query := `SELECT id, resume_builder_id, name, issuer, issue_date, expiry_date, url, sort_order, created_at, updated_at FROM resume_certifications WHERE id = $1 AND resume_builder_id = $2`
	c := &model.Certification{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&c.ID, &c.ResumeBuilderID, &c.Name, &c.Issuer, &c.IssueDate, &c.ExpiryDate, &c.URL, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return c, nil
}

// --- Projects ---

func (r *ResumeBuilderRepository) CreateProject(ctx context.Context, proj *model.Project) error {
	proj.ID = uuid.New().String()
	now := time.Now().UTC()
	proj.CreatedAt = now
	proj.UpdatedAt = now

	query := `INSERT INTO resume_projects (id, resume_builder_id, name, url, start_date, end_date, description, sort_order, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.q.Exec(ctx, query,
		proj.ID, proj.ResumeBuilderID, proj.Name, proj.URL, proj.StartDate, proj.EndDate, proj.Description, proj.SortOrder,
		proj.CreatedAt, proj.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) UpdateProject(ctx context.Context, proj *model.Project) error {
	query := `UPDATE resume_projects SET name = $1, url = $2, start_date = $3, end_date = $4, description = $5, sort_order = $6 WHERE id = $7 AND resume_builder_id = $8`
	tag, err := r.q.Exec(ctx, query, proj.Name, proj.URL, proj.StartDate, proj.EndDate, proj.Description, proj.SortOrder, proj.ID, proj.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_projects WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListProjects(ctx context.Context, resumeBuilderID string) ([]*model.Project, error) {
	query := `SELECT id, resume_builder_id, name, url, start_date, end_date, description, sort_order, created_at, updated_at FROM resume_projects WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Project
	for rows.Next() {
		p := &model.Project{}
		if err := rows.Scan(&p.ID, &p.ResumeBuilderID, &p.Name, &p.URL, &p.StartDate, &p.EndDate, &p.Description, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*model.Project, error) {
	query := `SELECT id, resume_builder_id, name, url, start_date, end_date, description, sort_order, created_at, updated_at FROM resume_projects WHERE id = $1 AND resume_builder_id = $2`
	p := &model.Project{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&p.ID, &p.ResumeBuilderID, &p.Name, &p.URL, &p.StartDate, &p.EndDate, &p.Description, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return p, nil
}

// --- Volunteering ---

func (r *ResumeBuilderRepository) CreateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	vol.ID = uuid.New().String()
	now := time.Now().UTC()
	vol.CreatedAt = now
	vol.UpdatedAt = now

	query := `INSERT INTO resume_volunteering (id, resume_builder_id, organization, role, start_date, end_date, description, sort_order, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.q.Exec(ctx, query,
		vol.ID, vol.ResumeBuilderID, vol.Organization, vol.Role, vol.StartDate, vol.EndDate, vol.Description, vol.SortOrder,
		vol.CreatedAt, vol.UpdatedAt,
	)
	return err
}

func (r *ResumeBuilderRepository) UpdateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	query := `UPDATE resume_volunteering SET organization = $1, role = $2, start_date = $3, end_date = $4, description = $5, sort_order = $6 WHERE id = $7 AND resume_builder_id = $8`
	tag, err := r.q.Exec(ctx, query, vol.Organization, vol.Role, vol.StartDate, vol.EndDate, vol.Description, vol.SortOrder, vol.ID, vol.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_volunteering WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error) {
	query := `SELECT id, resume_builder_id, organization, role, start_date, end_date, description, sort_order, created_at, updated_at FROM resume_volunteering WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.Volunteering
	for rows.Next() {
		v := &model.Volunteering{}
		if err := rows.Scan(&v.ID, &v.ResumeBuilderID, &v.Organization, &v.Role, &v.StartDate, &v.EndDate, &v.Description, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error) {
	query := `SELECT id, resume_builder_id, organization, role, start_date, end_date, description, sort_order, created_at, updated_at FROM resume_volunteering WHERE id = $1 AND resume_builder_id = $2`
	v := &model.Volunteering{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&v.ID, &v.ResumeBuilderID, &v.Organization, &v.Role, &v.StartDate, &v.EndDate, &v.Description, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return v, nil
}

// --- Custom Sections ---

func (r *ResumeBuilderRepository) CreateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	cs.ID = uuid.New().String()
	now := time.Now().UTC()
	cs.CreatedAt = now
	cs.UpdatedAt = now

	query := `INSERT INTO resume_custom_sections (id, resume_builder_id, title, content, sort_order, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.q.Exec(ctx, query, cs.ID, cs.ResumeBuilderID, cs.Title, cs.Content, cs.SortOrder, cs.CreatedAt, cs.UpdatedAt)
	return err
}

func (r *ResumeBuilderRepository) UpdateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	query := `UPDATE resume_custom_sections SET title = $1, content = $2, sort_order = $3 WHERE id = $4 AND resume_builder_id = $5`
	tag, err := r.q.Exec(ctx, query, cs.Title, cs.Content, cs.SortOrder, cs.ID, cs.ResumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	tag, err := r.q.Exec(ctx, `DELETE FROM resume_custom_sections WHERE id = $1 AND resume_builder_id = $2`, id, resumeBuilderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrSectionEntryNotFound
	}
	return nil
}

func (r *ResumeBuilderRepository) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error) {
	query := `SELECT id, resume_builder_id, title, content, sort_order, created_at, updated_at FROM resume_custom_sections WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.CustomSection
	for rows.Next() {
		cs := &model.CustomSection{}
		if err := rows.Scan(&cs.ID, &cs.ResumeBuilderID, &cs.Title, &cs.Content, &cs.SortOrder, &cs.CreatedAt, &cs.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, cs)
	}
	return items, rows.Err()
}

func (r *ResumeBuilderRepository) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error) {
	query := `SELECT id, resume_builder_id, title, content, sort_order, created_at, updated_at FROM resume_custom_sections WHERE id = $1 AND resume_builder_id = $2`
	cs := &model.CustomSection{}
	err := r.q.QueryRow(ctx, query, id, resumeBuilderID).Scan(
		&cs.ID, &cs.ResumeBuilderID, &cs.Title, &cs.Content, &cs.SortOrder, &cs.CreatedAt, &cs.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrSectionEntryNotFound
		}
		return nil, err
	}
	return cs, nil
}

// --- Section Order ---

func (r *ResumeBuilderRepository) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error {
	query := `
		INSERT INTO resume_section_orders (id, resume_builder_id, section_key, sort_order, is_visible, column_placement)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		ON CONFLICT (resume_builder_id, section_key)
		DO UPDATE SET sort_order = $3, is_visible = $4, column_placement = $5
	`
	for _, o := range orders {
		col := o.Column
		if col == "" {
			col = "main"
		}
		if _, err := r.q.Exec(ctx, query, resumeBuilderID, o.SectionKey, o.SortOrder, o.IsVisible, col); err != nil {
			return err
		}
	}
	return nil
}

func (r *ResumeBuilderRepository) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error) {
	query := `SELECT id, resume_builder_id, section_key, sort_order, is_visible, column_placement FROM resume_section_orders WHERE resume_builder_id = $1 ORDER BY sort_order`
	rows, err := r.q.Query(ctx, query, resumeBuilderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.SectionOrder
	for rows.Next() {
		o := &model.SectionOrder{}
		if err := rows.Scan(&o.ID, &o.ResumeBuilderID, &o.SectionKey, &o.SortOrder, &o.IsVisible, &o.Column); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}
