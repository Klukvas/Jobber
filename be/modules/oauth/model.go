// Package oauth implements the public PKCE authorization server for MCP.
package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUnsupportedGrant      = errors.New("unsupported_grant_type")
	ErrNotFound              = errors.New("oauth record not found")
	ErrInvalidGrant          = errors.New("invalid_grant")
	ErrInvalidClient         = errors.New("invalid_client")
	ErrInvalidRequest        = errors.New("invalid_request")
	ErrInvalidRedirectURI    = fmt.Errorf("%w: invalid_redirect_uri", ErrInvalidRequest)
	ErrInvalidClientMetadata = fmt.Errorf("%w: invalid_client_metadata", ErrInvalidRequest)
	ErrMetadataBusy          = errors.New("client metadata fetch capacity exhausted")
)

type Client struct {
	ID            string     `json:"client_id"`
	Name          string     `json:"client_name"`
	RedirectURIs  []string   `json:"redirect_uris"`
	GrantTypes    []string   `json:"grant_types"`
	ResponseTypes []string   `json:"response_types"`
	AuthMethod    string     `json:"token_endpoint_auth_method"`
	IssuedAt      int64      `json:"client_id_issued_at,omitempty"`
	IsMetadata    bool       `json:"-"`
	FetchedAt     *time.Time `json:"-"`
}

type AuthorizeParams struct {
	ResponseType    string `json:"response_type" form:"response_type"`
	ClientID        string `json:"client_id" form:"client_id"`
	RedirectURI     string `json:"redirect_uri" form:"redirect_uri"`
	Challenge       string `json:"code_challenge" form:"code_challenge"`
	ChallengeMethod string `json:"code_challenge_method" form:"code_challenge_method"`
	State           string `json:"state" form:"state"`
	Scope           string `json:"scope" form:"scope"`
	Resource        string `json:"resource" form:"resource"`
	Decision        string `json:"decision"`
}

type Preview struct {
	Client      *PreviewClient `json:"client,omitempty"`
	Scope       string         `json:"scope,omitempty"`
	Resource    string         `json:"resource,omitempty"`
	PlanAllowed bool           `json:"plan_allowed"`
	RedirectTo  string         `json:"redirect_to,omitempty"`
}

type PreviewClient struct {
	ID           string `json:"client_id"`
	Name         string `json:"name"`
	RedirectHost string `json:"redirect_host"`
	IsMetadata   bool   `json:"is_metadata_document"`
}

type Grant struct {
	ID           string     `json:"id"`
	ClientName   string     `json:"client_name"`
	ClientID     string     `json:"client_id"`
	RedirectHost string     `json:"redirect_host"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at"`
}

type Code struct {
	Hash, GrantID, RedirectURI, Challenge, Resource string
	ExpiresAt                                       time.Time
}

type ExchangeRequest struct {
	Kind, ClientID, Credential, RedirectURI, Verifier, Resource, Scope string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type TokenPair struct {
	AccessHash, RefreshHash, Resource string
	AccessExpiry, RefreshExpiry       time.Time
}

type Identity struct {
	UserID, GrantID string
	ExpiresAt       time.Time
}

type Store interface {
	GetClient(context.Context, string) (*Client, error)
	SaveClient(context.Context, Client) error
	CreateCode(context.Context, string, string, Code) error
	Exchange(context.Context, ExchangeRequest, TokenPair) error
	Authenticate(context.Context, string, string) (*Identity, error)
	RevokeToken(context.Context, string, string) error
	ListGrants(context.Context, string) ([]Grant, error)
	RevokeGrant(context.Context, string, string) error
	Cleanup(context.Context) error
}

type PaidPlanChecker interface {
	RequirePaidPlan(context.Context, string) error
}
type MetadataFetcher interface {
	Fetch(context.Context, string) (*Client, error)
}
