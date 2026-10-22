package interviewprep

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// guardDispatch serializes visibility/content edits with the actual request body
// write. It owns no user operation lock and releases before model response wait.
func (s *Store) guardDispatch(ctx context.Context, o Operation, snapshot Snapshot) (func() error, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	var once sync.Once
	var releaseErr error
	release := func() error {
		once.Do(func() {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancel()
			releaseErr = tx.Rollback(cleanup)
			if errors.Is(releaseErr, pgx.ErrTxClosed) {
				releaseErr = nil
			}
		})
		return releaseErr
	}
	if _, err = tx.Exec(ctx, `SET LOCAL lock_timeout='2s'`); err == nil {
		err = lockSource(ctx, tx, o.Owner, o.JobID, snapshot.ResumeID)
	}
	if err == nil {
		var current Snapshot
		current, err = readSnapshot(ctx, tx, o.Owner, o.JobID, snapshot.ResumeID, snapshot.Locale)
		if err == nil && current.Hash != snapshot.Hash {
			err = ErrSourceChanged
		}
	}
	if err != nil {
		_ = release()
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
			err = ErrSourceChanged
		}
		return nil, err
	}
	return release, nil
}
