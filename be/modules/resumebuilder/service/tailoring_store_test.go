package service

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisTailoringProposalStore(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewRedisTailoringProposalStore(client)
	_, r, _, _, _ := tailoringFixture(t)
	p := &model.TailoringProposal{TailoringProposalDTO: model.TailoringProposalDTO{ProposalID: "proposal", ExpiresAt: time.Now().Add(30 * time.Minute)}, UserID: "user", Source: r.source}
	require.NoError(t, store.Save(t.Context(), p))
	got, err := store.Get(t.Context(), "user", "proposal")
	require.NoError(t, err)
	assert.Equal(t, r.source, got.Source)
	_, err = store.Get(t.Context(), "foreign", "proposal")
	assert.ErrorIs(t, err, model.ErrProposalExpired)
	ttl := mr.TTL("resume-tailoring:user:proposal")
	p.AcceptedFingerprint = "accepted"
	require.NoError(t, store.Save(t.Context(), p))
	assert.LessOrEqual(t, mr.TTL("resume-tailoring:user:proposal"), ttl)
	mr.FastForward(31 * time.Minute)
	_, err = store.Get(t.Context(), "user", "proposal")
	assert.ErrorIs(t, err, model.ErrProposalExpired)
	mr.SetError("unavailable")
	assert.ErrorIs(t, store.Save(t.Context(), p), model.ErrProposalStoreUnavailable)
}
