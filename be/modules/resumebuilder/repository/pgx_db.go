package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPool is the subset of *pgxpool.Pool used for connection lifecycle
// (transactions and per-call connection acquisition). Both *pgxpool.Pool and
// pgxmock.PgxPoolIface satisfy it, enabling unit tests with a mock DB.
//
// Note: pgxmock's Acquire returns a "not implemented" error, so the
// Acquire-based GetFullResume happy path cannot be exercised with the mock;
// only its acquire-failure branch is covered in tests.
type PgxPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
}
