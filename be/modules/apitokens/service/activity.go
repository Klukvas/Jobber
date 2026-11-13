package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/apitokens/ports"
)

type ServiceOption func(*APITokenService)

func WithActivityRepository(repo ports.ActivityRepository) ServiceOption {
	return func(s *APITokenService) { s.activity = repo }
}

// ListActivity is available after downgrade and deliberately has no paid gate.
func (s *APITokenService) ListActivity(ctx context.Context, userID string, limit int) ([]*model.ActivityDTO, error) {
	if limit < 1 || limit > model.MaxActivityLimit {
		return nil, model.ErrInvalidActivityLimit
	}
	if s.activity == nil {
		return nil, errors.New("mcp activity repository unavailable")
	}
	items, err := s.activity.ListActivity(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list mcp activity: %w", err)
	}
	return items, nil
}
