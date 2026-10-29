package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
)

type manualRepository struct {
	*MockSubscriptionRepository
	manual *model.ManualAccess
	err    error
}

func (r *manualRepository) ManualAccess(context.Context, string) (*model.ManualAccess, error) {
	return r.manual, r.err
}
func TestManualAccessEnforcesAndReportsEffectivePlan(t *testing.T) {
	for _, status := range []string{"free", "cancelled", "paused", "active"} {
		t.Run(status, func(t *testing.T) {
			repo := &manualRepository{MockSubscriptionRepository: &MockSubscriptionRepository{GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return &model.Subscription{Plan: "free", Status: status}, nil
			}}, manual: &model.ManualAccess{ID: "manual", PeriodEnd: time.Now().Add(time.Hour)}}
			svc := NewSubscriptionService(repo, nil, BillingConfig{})
			require.NoError(t, svc.RequirePaidPlan(t.Context(), "user"))
			plan, err := svc.effectivePlan(t.Context(), "user")
			require.NoError(t, err)
			require.Equal(t, PlanPro, plan)
			for _, resource := range []string{"jobs", "resumes", "resume_builders", "cover_letters"} {
				limit, err := svc.ResourceLimit(t.Context(), "user", resource)
				require.NoError(t, err)
				require.NotZero(t, limit)
			}
			dto, err := svc.GetSubscription(t.Context(), "user")
			require.NoError(t, err)
			require.Equal(t, PlanPro, dto.Plan)
			require.True(t, dto.ManualOnly)
			require.Equal(t, repo.manual, dto.ManualAccess)
			repo.manual = nil
			require.ErrorIs(t, svc.RequirePaidPlan(t.Context(), "user"), model.ErrPaidFeature)
		})
	}
}
func TestManualRevocationPreservesEffectiveProviderPlan(t *testing.T) {
	repo := &manualRepository{MockSubscriptionRepository: &MockSubscriptionRepository{GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
		return &model.Subscription{Plan: PlanEnterprise, Status: StatusActive}, nil
	}}, manual: &model.ManualAccess{ID: "manual"}}
	svc := NewSubscriptionService(repo, nil, BillingConfig{})
	dto, err := svc.GetSubscription(t.Context(), "user")
	require.NoError(t, err)
	require.Equal(t, PlanEnterprise, dto.Plan)
	require.False(t, dto.ManualOnly)
	repo.manual = nil
	require.NoError(t, svc.RequirePaidPlan(t.Context(), "user"))
	plan, err := svc.effectivePlan(t.Context(), "user")
	require.NoError(t, err)
	require.Equal(t, PlanEnterprise, plan)
	repo.err = errors.New("manual store unavailable")
	require.Error(t, svc.RequirePaidPlan(t.Context(), "user"), "manual lookup failures must fail closed")
}
