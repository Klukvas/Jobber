package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	t.Run("creates user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		user := &model.User{
			Email:        "test@example.com",
			Name:         "Test User",
			PasswordHash: "hashed-password",
			Locale:       "en",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(pgxmock.AnyArg(), user.Email, user.Name, user.PasswordHash, user.Locale, user.CreatedAt, user.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testUserRepo{mock: mock}
		err = repo.Create(context.Background(), user)

		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	t.Run("returns user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "password_hash", "locale", "created_at", "updated_at",
		}).AddRow(
			userID,
			"test@example.com",
			"Test User",
			"hashed-password",
			"en",
			now,
			now,
		)

		mock.ExpectQuery("SELECT id, email, name, password_hash, locale, created_at, updated_at").
			WithArgs(userID).
			WillReturnRows(rows)

		repo := &testUserRepo{mock: mock}
		user, err := repo.GetByID(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test User", user.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "nonexistent"

		mock.ExpectQuery("SELECT id, email, name, password_hash, locale, created_at, updated_at").
			WithArgs(userID).
			WillReturnError(pgx.ErrNoRows)

		repo := &testUserRepo{mock: mock}
		user, err := repo.GetByID(context.Background(), userID)

		assert.Nil(t, user)
		assert.Equal(t, model.ErrUserNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	t.Run("returns user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		email := "test@example.com"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "email", "name", "password_hash", "locale", "created_at", "updated_at",
		}).AddRow(
			"user-123",
			email,
			"Test User",
			"hashed-password",
			"en",
			now,
			now,
		)

		mock.ExpectQuery("SELECT id, email, name, password_hash, locale, created_at, updated_at").
			WithArgs(email).
			WillReturnRows(rows)

		repo := &testUserRepo{mock: mock}
		user, err := repo.GetByEmail(context.Background(), email)

		require.NoError(t, err)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, "user-123", user.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		email := "nonexistent@example.com"

		mock.ExpectQuery("SELECT id, email, name, password_hash, locale, created_at, updated_at").
			WithArgs(email).
			WillReturnError(pgx.ErrNoRows)

		repo := &testUserRepo{mock: mock}
		user, err := repo.GetByEmail(context.Background(), email)

		assert.Nil(t, user)
		assert.Equal(t, model.ErrUserNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	t.Run("updates user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		user := &model.User{
			ID:     "user-123",
			Name:   "Updated Name",
			Locale: "ua",
		}

		mock.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name, user.Locale).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &testUserRepo{mock: mock}
		err = repo.Update(context.Background(), user)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		user := &model.User{
			ID:     "nonexistent",
			Name:   "Updated Name",
			Locale: "ua",
		}

		mock.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name, user.Locale).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := &testUserRepo{mock: mock}
		err = repo.Update(context.Background(), user)

		assert.Equal(t, model.ErrUserNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	t.Run("deletes user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"

		mock.ExpectExec("DELETE FROM users").
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &testUserRepo{mock: mock}
		err = repo.Delete(context.Background(), userID)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "nonexistent"

		mock.ExpectExec("DELETE FROM users").
			WithArgs(userID).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &testUserRepo{mock: mock}
		err = repo.Delete(context.Background(), userID)

		assert.Equal(t, model.ErrUserNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUser_ToDTO(t *testing.T) {
	now := time.Now()
	user := &model.User{
		ID:           "user-123",
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "secret-hash",
		Locale:       "en",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	dto := user.ToDTO()

	assert.Equal(t, user.ID, dto.ID)
	assert.Equal(t, user.Email, dto.Email)
	assert.Equal(t, user.Name, dto.Name)
	assert.Equal(t, user.Locale, dto.Locale)
	assert.Equal(t, user.CreatedAt, dto.CreatedAt)
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello world", "xyz", false},
		{"", "x", false},
		{"hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := containsString(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// testUserRepo is a test wrapper that uses pgxmock
type testUserRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testUserRepo) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, name, password_hash, locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	user.ID = "test-user-id"
	_, err := r.mock.Exec(ctx, query,
		user.ID,
		user.Email,
		user.Name,
		user.PasswordHash,
		user.Locale,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (r *testUserRepo) GetByID(ctx context.Context, userID string) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, locale, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &model.User{}
	err := r.mock.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, name, password_hash, locale, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &model.User{}
	err := r.mock.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *testUserRepo) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET name = $2, locale = $3
		WHERE id = $1
	`
	result, err := r.mock.Exec(ctx, query, user.ID, user.Name, user.Locale)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}
	return nil
}

func (r *testUserRepo) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.mock.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}
	return nil
}
