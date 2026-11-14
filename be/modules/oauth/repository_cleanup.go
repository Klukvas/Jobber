package oauth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) Cleanup(ctx context.Context) error {
	// Keep replay tombstones for 30 days beyond expiry. Bound each sweep and let
	// SKIP LOCKED avoid delaying ongoing exchanges; cascade removal covers users.
	_, err := r.db.Exec(ctx, `DELETE FROM oauth_authorization_codes WHERE code_hash IN
 (SELECT code_hash FROM oauth_authorization_codes WHERE expires_at < now()-interval '30 days' LIMIT 100 FOR UPDATE SKIP LOCKED)`)
	if err != nil {
		return fmt.Errorf("clean oauth codes: %w", err)
	}
	_, err = r.db.Exec(ctx, `DELETE FROM oauth_tokens WHERE token_hash IN
 (SELECT token_hash FROM oauth_tokens WHERE (kind='access' AND expires_at<now())
 OR expires_at<now()-interval '30 days' LIMIT 100 FOR UPDATE SKIP LOCKED)`)
	if err != nil {
		return fmt.Errorf("clean oauth tokens: %w", err)
	}
	return r.cleanupClients(ctx)
}

// Lock clients before pruning their revoked grants: new consent cannot insert a
// referencing grant until this transaction finishes. DCR records never expire.
func (r *Repository) cleanupClients(ctx context.Context) error {
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT c.client_id FROM oauth_clients c
 WHERE c.is_metadata_document AND (c.fetched_at IS NULL OR c.fetched_at < now()-interval '24 hours')
 AND NOT EXISTS (SELECT 1 FROM oauth_grants g WHERE g.client_id=c.client_id AND g.revoked_at IS NULL)
 ORDER BY c.fetched_at NULLS FIRST LIMIT 100 FOR UPDATE OF c SKIP LOCKED`)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil || len(ids) == 0 {
			return err
		}
		// The FK does not cascade from clients. Bound grant removal too, and let
		// subsequent sweeps finish clients with more than 100 historical grants.
		_, err = tx.Exec(ctx, `DELETE FROM oauth_grants WHERE id IN
 (SELECT id FROM oauth_grants WHERE client_id=ANY($1) AND revoked_at IS NOT NULL
 LIMIT 100 FOR UPDATE SKIP LOCKED)`, ids)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM oauth_clients c WHERE c.client_id=ANY($1)
 AND NOT EXISTS (SELECT 1 FROM oauth_grants g WHERE g.client_id=c.client_id)`, ids)
		return err
	})
	if err != nil {
		return fmt.Errorf("clean oauth metadata clients: %w", err)
	}
	return nil
}
