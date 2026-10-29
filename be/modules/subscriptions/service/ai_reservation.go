package service

import (
	"context"
	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"log/slog"
	"sync"
	"time"
)

type reservationScopeKey struct{}
type reservationScope struct {
	mu              sync.Mutex
	owner           string
	receipt         string
	reserved        bool
	consumed        bool
	commitAttempted bool
}

// AIRequestContext tracks an atomic quota reservation through legacy AI service
// calls. Install once per HTTP request; it carries no user content.
func AIRequestContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, reservationScopeKey{}, &reservationScope{})
}
func (s *SubscriptionService) reserveScopedAI(ctx context.Context, owner string) (bool, error) {
	scope, ok := ctx.Value(reservationScopeKey{}).(*reservationScope)
	if !ok {
		return false, nil
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if scope.reserved {
		if scope.owner != owner {
			return true, context.Canceled
		}
		return true, nil
	}
	repo, ok := s.repo.(interface {
		ReserveAI(context.Context, string, int) (string, error)
	})
	if !ok {
		return true, context.Canceled
	}
	limit, err := s.ResourceLimit(ctx, owner, "ai")
	if err != nil {
		return true, err
	}
	id, err := repo.ReserveAI(ctx, owner, limit)
	if err != nil {
		return true, err
	}
	scope.receipt = id
	scope.owner = owner
	scope.reserved = true
	return true, nil
}
func (s *SubscriptionService) consumeScopedAI(ctx context.Context, owner, usageType string) (bool, error) {
	scope, ok := ctx.Value(reservationScopeKey{}).(*reservationScope)
	if !ok {
		return false, nil
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if !scope.reserved || scope.owner != owner {
		return false, nil
	}
	if scope.consumed {
		return true, nil
	}
	repo, ok := s.repo.(interface {
		CommitAIReservation(context.Context, string, string, string) error
	})
	if !ok {
		return true, context.Canceled
	}
	commit, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	scope.commitAttempted = true
	if err := repo.CommitAIReservation(commit, owner, scope.receipt, usageType); err != nil {
		slog.Error("ai quota reservation commit failed", "outcome", "save_failed")
		return true, err
	}
	scope.consumed = true
	return true, nil
}

// FinishAIRequest releases reservations for rejected inputs, ownership failures,
// and unsuccessful legacy calls. Interview operations own their separate receipt.
func (s *SubscriptionService) FinishAIRequest(ctx context.Context) {
	if ledger, ok := s.repo.(interface {
		FinishCreditOperation(context.Context, string, string, ...model.CreditOperation) error
	}); ok {
		op, durable, found := ai.CurrentCreditOperation(ctx)
		if !found || durable {
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if err := ledger.FinishCreditOperation(cleanup, op.UserID, op.Key, op); err != nil {
			slog.Error("finish ai credit operation failed", "outcome", "save_failed")
		}
		return
	}
	scope, ok := ctx.Value(reservationScopeKey{}).(*reservationScope)
	if !ok {
		return
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if !scope.reserved || scope.consumed || scope.commitAttempted {
		return
	}
	repo, ok := s.repo.(interface {
		ReleaseAI(context.Context, string, string) error
	})
	if !ok {
		return
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := repo.ReleaseAI(cleanup, scope.owner, scope.receipt); err != nil {
		slog.Error("ai quota reservation release failed", "outcome", "save_failed")
	}
}
