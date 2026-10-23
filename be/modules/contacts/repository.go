package contacts

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ db Database }

func NewRepository(db Database) *Repository { return &Repository{db: db} }

const columns = `c.id,c.job_id,c.name,c.role,c.email,c.telegram,c.linkedin,c.note,c.created_at,c.updated_at`

func scanContact(row pgx.Row) (*Contact, error) {
	var c Contact
	err := row.Scan(&c.ID, &c.JobID, &c.Name, &c.Role, &c.Email, &c.Telegram, &c.LinkedIn, &c.Note, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read contact: %w", err)
	}
	return &c, nil
}
func (r *Repository) List(ctx context.Context, user, job string) ([]Contact, error) {
	var owned bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE id=$1 AND user_id=$2)`, job, user).Scan(&owned); err != nil {
		return nil, fmt.Errorf("check contact parent: %w", err)
	}
	if !owned {
		return nil, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT `+columns+` FROM job_contacts c JOIN jobs j ON j.id=c.job_id WHERE c.job_id=$1 AND j.user_id=$2 ORDER BY c.created_at,c.id`, job, user)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()
	result := []Contact{}
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contacts: %w", err)
	}
	return result, nil
}
func (r *Repository) Create(ctx context.Context, user, job string, f Fields) (*Contact, error) {
	return scanContact(r.db.QueryRow(ctx, `INSERT INTO job_contacts AS c(id,job_id,name,role,email,telegram,linkedin,note) SELECT $1,$2,$3,$4,$5,$6,$7,$8 FROM jobs WHERE id=$2 AND user_id=$9 RETURNING `+columns, uuid.NewString(), job, f.Name, f.Role, f.Email, f.Telegram, f.LinkedIn, f.Note, user))
}
func (r *Repository) Update(ctx context.Context, user, job, id string, f Fields) (*Contact, error) {
	return scanContact(r.db.QueryRow(ctx, `UPDATE job_contacts c SET name=$4,role=$5,email=$6,telegram=$7,linkedin=$8,note=$9,updated_at=now() FROM jobs j WHERE c.id=$1 AND c.job_id=$2 AND j.id=c.job_id AND j.user_id=$3 RETURNING `+columns, id, job, user, f.Name, f.Role, f.Email, f.Telegram, f.LinkedIn, f.Note))
}
func (r *Repository) Delete(ctx context.Context, user, job, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM job_contacts c USING jobs j WHERE c.id=$1 AND c.job_id=$2 AND j.id=c.job_id AND j.user_id=$3`, id, job, user)
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
