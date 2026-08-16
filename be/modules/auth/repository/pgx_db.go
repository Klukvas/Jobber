package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PgxDB is the subset of *pgxpool.Pool the auth repositories use. Both
// *pgxpool.Pool and pgxmock.PgxPoolIface satisfy it, enabling unit tests with
// a mock DB.
type PgxDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
