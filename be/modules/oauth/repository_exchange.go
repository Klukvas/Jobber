package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type credential struct {
	grantID, clientID, resource, redirect, challenge string
	expires                                          time.Time
	used, revoked                                    *time.Time
}

func (r *Repository) Exchange(ctx context.Context, request ExchangeRequest, pair TokenPair) error {
	var rejected bool
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		record, err := lockCredential(ctx, tx, request)
		if err != nil {
			return err
		}
		if record.clientID != request.ClientID {
			return ErrInvalidGrant
		}
		if record.used != nil {
			rejected = true
			return revokeGrant(ctx, tx, record.grantID)
		}
		if err := validateCredential(record, request, pair.Resource); err != nil {
			return err
		}
		if err := consumeCredential(ctx, tx, request); err != nil {
			return err
		}
		return insertPair(ctx, tx, record.grantID, pair)
	})
	if rejected && err == nil {
		return ErrInvalidGrant
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidGrant
	}
	if err != nil {
		return fmt.Errorf("exchange oauth credential: %w", err)
	}
	return nil
}

func lockCredential(ctx context.Context, tx pgx.Tx, request ExchangeRequest) (*credential, error) {
	var grantID string
	hash := hashSecret(request.Credential)
	query := `SELECT g.id FROM oauth_grants g JOIN oauth_authorization_codes c ON c.grant_id=g.id WHERE c.code_hash=$1 FOR UPDATE OF g`
	if request.Kind == "refresh_token" {
		query = `SELECT g.id FROM oauth_grants g JOIN oauth_tokens t ON t.grant_id=g.id WHERE t.token_hash=$1 AND t.kind='refresh' FOR UPDATE OF g`
	}
	if err := tx.QueryRow(ctx, query, hash).Scan(&grantID); err != nil {
		return nil, err
	}
	record := &credential{grantID: grantID}
	// A second statement sees state committed by a concurrent exchange after the
	// grant lock was acquired. Every exchange/revocation takes this same lock.
	if request.Kind == "authorization_code" {
		err := tx.QueryRow(ctx, `SELECT g.client_id,g.revoked_at,c.resource,c.expires_at,c.used_at,c.redirect_uri,c.code_challenge
 FROM oauth_authorization_codes c JOIN oauth_grants g ON g.id=c.grant_id WHERE c.code_hash=$1`, hash).
			Scan(&record.clientID, &record.revoked, &record.resource, &record.expires, &record.used, &record.redirect, &record.challenge)
		return record, err
	}
	err := tx.QueryRow(ctx, `SELECT g.client_id,COALESCE(g.revoked_at,t.revoked_at),t.resource,t.expires_at,t.rotated_at
 FROM oauth_tokens t JOIN oauth_grants g ON g.id=t.grant_id WHERE t.token_hash=$1 AND t.kind='refresh'`, hash).
		Scan(&record.clientID, &record.revoked, &record.resource, &record.expires, &record.used)
	return record, err
}

func validateCredential(record *credential, request ExchangeRequest, resource string) error {
	if record.revoked != nil || !record.expires.After(time.Now()) || record.resource != resource {
		return ErrInvalidGrant
	}
	if request.Kind == "authorization_code" && (record.redirect != request.RedirectURI || !verifyPKCE(request.Verifier, record.challenge)) {
		return ErrInvalidGrant
	}
	return nil
}

func consumeCredential(ctx context.Context, tx pgx.Tx, request ExchangeRequest) error {
	query := `UPDATE oauth_authorization_codes SET used_at=now() WHERE code_hash=$1`
	if request.Kind == "refresh_token" {
		query = `UPDATE oauth_tokens SET rotated_at=now() WHERE token_hash=$1`
	}
	_, err := tx.Exec(ctx, query, hashSecret(request.Credential))
	return err
}

func insertPair(ctx context.Context, tx pgx.Tx, grantID string, pair TokenPair) error {
	_, err := tx.Exec(ctx, `INSERT INTO oauth_tokens (token_hash,kind,grant_id,resource,scope,expires_at)
 VALUES ($1,'access',$3,$4,'mcp',$5),($2,'refresh',$3,$4,'mcp',$6)`, pair.AccessHash, pair.RefreshHash, grantID, pair.Resource, pair.AccessExpiry, pair.RefreshExpiry)
	return err
}

func (r *Repository) Authenticate(ctx context.Context, hash, resource string) (*Identity, error) {
	var identity Identity
	err := r.db.QueryRow(ctx, `WITH valid AS (
 SELECT g.user_id,g.id,t.expires_at FROM oauth_tokens t JOIN oauth_grants g ON g.id=t.grant_id
 JOIN users u ON u.id=g.user_id WHERE t.token_hash=$1 AND t.kind='access' AND t.resource=$2
 AND t.expires_at>now() AND t.revoked_at IS NULL AND g.revoked_at IS NULL
 ), touched AS (UPDATE oauth_grants SET last_used_at=now() WHERE id IN (SELECT id FROM valid)
 AND (last_used_at IS NULL OR last_used_at<now()-interval '1 minute'))
 SELECT user_id,id,expires_at FROM valid`, hash, resource).Scan(&identity.UserID, &identity.GrantID, &identity.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate oauth token: %w", err)
	}
	return &identity, nil
}

func (r *Repository) RevokeToken(ctx context.Context, hash, clientID string) error {
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		var grantID string
		err := tx.QueryRow(ctx, `SELECT g.id FROM oauth_grants g JOIN oauth_tokens t ON t.grant_id=g.id
 WHERE t.token_hash=$1 AND ($2='' OR g.client_id=$2) FOR UPDATE OF g`, hash, clientID).Scan(&grantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return revokeGrant(ctx, tx, grantID)
	})
	if err != nil {
		return fmt.Errorf("revoke oauth token: %w", err)
	}
	return nil
}
