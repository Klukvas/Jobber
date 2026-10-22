package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/redis/go-redis/v9"
)

type TailoringProposalStore interface {
	Save(context.Context, *model.TailoringProposal) error
	Get(context.Context, string, string) (*model.TailoringProposal, error)
}

type RedisTailoringProposalStore struct{ client *redis.Client }

func NewRedisTailoringProposalStore(client *redis.Client) *RedisTailoringProposalStore {
	return &RedisTailoringProposalStore{client: client}
}

func (s *RedisTailoringProposalStore) Save(ctx context.Context, proposal *model.TailoringProposal) error {
	if s == nil || s.client == nil {
		return model.ErrProposalStoreUnavailable
	}
	ttl := time.Until(proposal.ExpiresAt)
	if ttl <= 0 {
		return model.ErrProposalExpired
	}
	data, err := json.Marshal(proposal)
	if err != nil {
		return model.ErrProposalStoreUnavailable
	}
	if err := s.client.Set(ctx, "resume-tailoring:"+proposal.UserID+":"+proposal.ProposalID, data, ttl).Err(); err != nil {
		return model.ErrProposalStoreUnavailable
	}
	return nil
}

func (s *RedisTailoringProposalStore) Get(ctx context.Context, userID, proposalID string) (*model.TailoringProposal, error) {
	if s == nil || s.client == nil {
		return nil, model.ErrProposalStoreUnavailable
	}
	data, err := s.client.Get(ctx, "resume-tailoring:"+userID+":"+proposalID).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, model.ErrProposalExpired
	}
	if err != nil {
		return nil, model.ErrProposalStoreUnavailable
	}
	var proposal model.TailoringProposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return nil, model.ErrProposalStoreUnavailable
	}
	if proposal.UserID != userID || proposal.ProposalID != proposalID {
		return nil, model.ErrTailoringNotFound
	}
	if !time.Now().Before(proposal.ExpiresAt) {
		return nil, model.ErrProposalExpired
	}
	return &proposal, nil
}
