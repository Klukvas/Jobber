package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ db DB }

func NewRepository(db DB) *Repository { return &Repository{db: db} }

func (r *Repository) GetClient(ctx context.Context, id string) (*Client, error) {
	var client Client
	err := r.db.QueryRow(ctx, `SELECT client_id, client_name, redirect_uris, grant_types,
 response_types, is_metadata_document, fetched_at, extract(epoch FROM created_at)::bigint
 FROM oauth_clients WHERE client_id = $1`, id).Scan(&client.ID, &client.Name, &client.RedirectURIs, &client.GrantTypes, &client.ResponseTypes, &client.IsMetadata, &client.FetchedAt, &client.IssuedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get oauth client: %w", err)
	}
	client.AuthMethod = "none"
	return &client, nil
}

func (r *Repository) SaveClient(ctx context.Context, client Client) error {
	_, err := r.db.Exec(ctx, `INSERT INTO oauth_clients
 (client_id, client_name, redirect_uris, grant_types, response_types, is_metadata_document, fetched_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (client_id) DO UPDATE SET
 client_name = EXCLUDED.client_name, redirect_uris = EXCLUDED.redirect_uris,
 grant_types = EXCLUDED.grant_types, response_types = EXCLUDED.response_types, fetched_at = EXCLUDED.fetched_at
 WHERE oauth_clients.is_metadata_document AND EXCLUDED.is_metadata_document`,
		client.ID, client.Name, client.RedirectURIs, client.GrantTypes, client.ResponseTypes, client.IsMetadata, client.FetchedAt)
	if err != nil {
		return fmt.Errorf("save oauth client: %w", err)
	}
	return nil
}

func (r *Repository) CreateCode(ctx context.Context, userID, clientID string, code Code) error {
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		var grantID string
		err := tx.QueryRow(ctx, `INSERT INTO oauth_grants (id,user_id,client_id,scope,redirect_uri)
 VALUES ($1,$2,$3,'mcp',$4) ON CONFLICT (user_id,client_id) WHERE revoked_at IS NULL
 DO UPDATE SET redirect_uri = EXCLUDED.redirect_uri RETURNING id`, code.GrantID, userID, clientID, code.RedirectURI).Scan(&grantID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO oauth_authorization_codes
 (code_hash,grant_id,redirect_uri,code_challenge,resource,expires_at) VALUES ($1,$2,$3,$4,$5,$6)`,
			code.Hash, grantID, code.RedirectURI, code.Challenge, code.Resource, code.ExpiresAt)
		return err
	})
	if err != nil {
		return fmt.Errorf("create oauth code: %w", err)
	}
	return nil
}

func (r *Repository) ListGrants(ctx context.Context, userID string) ([]Grant, error) {
	rows, err := r.db.Query(ctx, `SELECT g.id, c.client_name, g.client_id, g.redirect_uri, g.created_at, g.last_used_at
 FROM oauth_grants g JOIN oauth_clients c ON c.client_id = g.client_id
 WHERE g.user_id = $1 AND g.revoked_at IS NULL ORDER BY g.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list oauth grants: %w", err)
	}
	defer rows.Close()
	grants := make([]Grant, 0)
	for rows.Next() {
		var grant Grant
		var redirect string
		if err := rows.Scan(&grant.ID, &grant.ClientName, &grant.ClientID, &redirect, &grant.CreatedAt, &grant.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan oauth grant: %w", err)
		}
		parsed, err := url.Parse(redirect)
		if err != nil {
			return nil, fmt.Errorf("parse stored oauth redirect: %w", err)
		}
		grant.RedirectHost = parsed.Host
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth grants: %w", err)
	}
	return grants, nil
}

func (r *Repository) RevokeGrant(ctx context.Context, userID, id string) error {
	err := r.transaction(ctx, func(tx pgx.Tx) error {
		var grantID string
		err := tx.QueryRow(ctx, `SELECT id FROM oauth_grants WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID).Scan(&grantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return revokeGrant(ctx, tx, grantID)
	})
	if err != nil {
		return fmt.Errorf("revoke oauth grant: %w", err)
	}
	return nil
}

func revokeGrant(ctx context.Context, tx pgx.Tx, id string) error {
	if _, err := tx.Exec(ctx, `UPDATE oauth_grants SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1`, id); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE oauth_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE grant_id=$1`, id)
	return err
}
