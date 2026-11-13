package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
)

type APITokenRepository interface {
	// Create atomically enforces MaxActiveTokens, counting only non-revoked, unexpired tokens.
	Create(ctx context.Context, token *model.APIToken) (*model.APIToken, error)
	List(ctx context.Context, userID string) ([]*model.APIToken, error)
	GetByHash(ctx context.Context, hash string) (*model.APIToken, error)
	Revoke(ctx context.Context, userID, tokenID string) error
	TouchLastUsed(ctx context.Context, tokenID string) error
}
