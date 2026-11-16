package oauth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
)

const maxConcurrentMetadataFetches = 4

type Service struct {
	store         Store
	paid          PaidPlanChecker
	fetcher       MetadataFetcher
	issuer        string
	metadataSlots chan struct{}
}

func NewService(store Store, paid PaidPlanChecker, issuer string, fetcher MetadataFetcher) *Service {
	if fetcher == nil {
		fetcher = NewCIMDFetcher(nil, nil)
	}
	return &Service{store: store, paid: paid, issuer: strings.TrimRight(issuer, "/"), fetcher: fetcher, metadataSlots: make(chan struct{}, maxConcurrentMetadataFetches)}
}

func (s *Service) Resource() string { return s.issuer + "/api/mcp" }

func (s *Service) Register(ctx context.Context, input Client) (*Client, error) {
	client, err := ValidateClient(input)
	if err != nil {
		return nil, err
	}
	client.ID, err = randomSecret("jbr_client_", 24)
	if err != nil {
		return nil, err
	}
	client.IsMetadata, client.FetchedAt = false, nil
	client.IssuedAt = time.Now().Unix()
	if err := s.store.SaveClient(ctx, client); err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *Service) client(ctx context.Context, id string, allowFetch bool) (*Client, bool, error) {
	if len(id) == 0 || len(id) > 2048 {
		return nil, false, ErrInvalidClient
	}
	client, err := s.store.GetClient(ctx, id)
	if err == nil && (!client.IsMetadata || client.FetchedAt != nil && time.Since(*client.FetchedAt) < 24*time.Hour) {
		return client, false, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	if !validMetadataURL(id) {
		return nil, false, ErrInvalidClient
	}
	if !allowFetch {
		return nil, false, subModel.ErrPaidFeature
	}
	select {
	case s.metadataSlots <- struct{}{}:
		defer func() { <-s.metadataSlots }()
	default:
		return nil, false, ErrMetadataBusy
	}
	client, err = s.fetcher.Fetch(ctx, id)
	if err != nil {
		return nil, false, ErrInvalidClient
	}
	return client, true, nil
}

func (s *Service) Preview(ctx context.Context, userID string, params AuthorizeParams) (*Preview, error) {
	// Reject syntactically invalid callbacks before even resolving a client.
	if ValidateRedirectURI(params.RedirectURI) != nil {
		return nil, ErrInvalidRequest
	}
	planErr := s.paid.RequirePaidPlan(ctx, userID)
	if planErr != nil && !errors.Is(planErr, subModel.ErrPaidFeature) {
		return nil, planErr
	}
	allowed := planErr == nil
	client, fetched, err := s.client(ctx, params.ClientID, allowed)
	if err != nil {
		return nil, err
	}
	// Never emit redirect_to or persist fetched metadata before trusting this callback.
	if !redirectAllowed(client.RedirectURIs, params.RedirectURI) {
		return nil, ErrInvalidRequest
	}
	if code := s.authorizationError(params, client); code != "" {
		return &Preview{RedirectTo: s.redirect(params, "error", code)}, nil
	}
	if fetched {
		if err := s.store.Cleanup(ctx); err != nil {
			return nil, err
		}
		if err := s.store.SaveClient(ctx, *client); err != nil {
			return nil, err
		}
	}
	redirect, _ := url.Parse(params.RedirectURI)
	return &Preview{Client: &PreviewClient{ID: client.ID, Name: client.Name, RedirectHost: redirect.Host, IsMetadata: client.IsMetadata}, Scope: "mcp", Resource: s.Resource(), PlanAllowed: allowed}, nil
}

func (s *Service) authorizationError(params AuthorizeParams, client *Client) string {
	switch {
	case params.ResponseType != "code":
		return "unsupported_response_type"
	case !contains(client.GrantTypes, "authorization_code") || !contains(client.ResponseTypes, "code"):
		return "unauthorized_client"
	case params.ChallengeMethod != "S256" || !validChallenge(params.Challenge):
		return "invalid_request"
	case len(params.State) > 2048:
		return "invalid_request"
	case params.Scope != "" && params.Scope != "mcp":
		return "invalid_scope"
	case params.Resource != "" && params.Resource != s.Resource():
		return "invalid_target"
	default:
		return ""
	}
}

func (s *Service) Authorize(ctx context.Context, userID string, params AuthorizeParams) (string, error) {
	if params.Decision != "allow" && params.Decision != "deny" {
		return "", ErrInvalidRequest
	}
	preview, err := s.Preview(ctx, userID, params)
	if err != nil {
		return "", err
	}
	if preview.RedirectTo != "" {
		return preview.RedirectTo, nil
	}
	if params.Decision == "deny" {
		return s.redirect(params, "error", "access_denied"), nil
	}
	if !preview.PlanAllowed {
		return "", subModel.ErrPaidFeature
	}
	secret, err := randomSecret("", 32)
	if err != nil {
		return "", err
	}
	code := Code{Hash: hashSecret(secret), GrantID: uuid.NewString(), RedirectURI: params.RedirectURI, Challenge: params.Challenge, Resource: s.Resource(), ExpiresAt: time.Now().UTC().Add(10 * time.Minute)}
	if err := s.store.Cleanup(ctx); err != nil {
		return "", err
	}
	if err := s.store.CreateCode(ctx, userID, params.ClientID, code); err != nil {
		return "", err
	}
	return s.redirect(params, "code", secret), nil
}

func (s *Service) redirect(params AuthorizeParams, key, value string) string {
	target, _ := url.Parse(params.RedirectURI)
	query := target.Query()
	// A registered URI's query must not inject a second OAuth result.
	for _, field := range []string{"code", "error", "state", "iss"} {
		query.Del(field)
	}
	query.Set(key, value)
	if params.State != "" {
		query.Set("state", params.State)
	}
	query.Set("iss", s.issuer)
	target.RawQuery = query.Encode()
	return target.String()
}
