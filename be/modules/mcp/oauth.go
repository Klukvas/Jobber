package mcp

import (
	"context"
	"strings"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/oauth"
)

type OAuthAuthenticator interface {
	Authenticate(context.Context, string) (*oauth.Identity, error)
}

type combinedAuthenticator struct {
	pat   TokenAuthenticator
	oauth OAuthAuthenticator
}

func NewTokenAuthenticator(pat TokenAuthenticator, oauth OAuthAuthenticator) TokenAuthenticator {
	return &combinedAuthenticator{pat: pat, oauth: oauth}
}

func (a *combinedAuthenticator) AuthenticateWithInfo(ctx context.Context, token string) (*patModel.AuthenticationInfo, error) {
	if strings.HasPrefix(token, "jbr_oat_") {
		identity, err := a.oauth.Authenticate(ctx, token)
		if err != nil {
			return nil, err
		}
		if identity == nil {
			return nil, patModel.ErrInvalidToken
		}
		return &patModel.AuthenticationInfo{UserID: identity.UserID, OAuthGrantID: identity.GrantID, ExpiresAt: identity.ExpiresAt}, nil
	}
	return a.pat.AuthenticateWithInfo(ctx, token)
}
