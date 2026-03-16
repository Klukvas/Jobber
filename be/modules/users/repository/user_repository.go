package repository

import (
	"context"
	"errors"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implements ports.UserRepository
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, name, password_hash, locale, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	user.ID = uuid.New().String()

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.Locale,
		user.EmailVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (PostgreSQL error code 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.ErrUserAlreadyExists
		}
		return err
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, locale, email_verified, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Locale,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, locale, email_verified, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Locale,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET name = $2, locale = $3
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, user.ID, user.Name, user.Locale)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// SetEmailVerified marks a user's email as verified
func (r *UserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	query := `UPDATE users SET email_verified = true WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

// UpdatePasswordHash updates the password hash for a user
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, userID, hash)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}

	return nil
}

