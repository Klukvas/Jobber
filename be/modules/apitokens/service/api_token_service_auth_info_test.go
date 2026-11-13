package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/stretchr/testify/require"
)

type countingAuthRepo struct {
	*fakeRepo
	lookups int
}

func (r *countingAuthRepo) GetByHash(ctx context.Context, hash string) (*model.APIToken, error) {
	r.lookups++
	return r.fakeRepo.GetByHash(ctx, hash)
}

func TestAPITokenService_AuthenticateWithInfo(t *testing.T) {
	plaintext, err := generateToken()
	require.NoError(t, err)
	expiry := time.Now().Add(time.Hour)
	for _, expiration := range []*time.Time{nil, &expiry} {
		t.Run("expiration", func(t *testing.T) {
			stored := &model.APIToken{ID: "token", UserID: "owner", ExpiresAt: expiration, TokenHash: hashToken(plaintext)}
			repo := &countingAuthRepo{fakeRepo: &fakeRepo{tokens: []*model.APIToken{stored}}}
			svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
			info, err := svc.AuthenticateWithInfo(t.Context(), plaintext)
			require.NoError(t, err)
			require.Equal(t, 1, repo.lookups)
			require.Equal(t, "owner", info.UserID)
			require.Equal(t, "token", info.TokenID)
			if expiration == nil {
				require.True(t, info.ExpiresAt.IsZero())
			} else {
				require.Equal(t, *expiration, info.ExpiresAt)
				info.ExpiresAt = info.ExpiresAt.Add(time.Hour)
				require.Equal(t, expiry, *stored.ExpiresAt, "returned identity must not mutate the stored token")
			}
		})
	}
}
