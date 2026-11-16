//go:build integration

package interviewprep

import (
	"context"
	"fmt"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type chargedInterviewFixtureClient struct {
	client  *testClient
	repo    *subRepo.SubscriptionRepository
	invalid bool
	calls   int
}

func (c *chargedInterviewFixtureClient) Interview(ctx context.Context, kind string, raw []byte, release func() error) ([]byte, error) {
	c.calls++
	op, _, ok := ai.CurrentCreditOperation(ctx)
	if !ok {
		return nil, fmt.Errorf("missing credit identity")
	}
	id, err := c.repo.ReserveCreditAttempt(ctx, subModel.AIAttempt{Operation: op, Stage: fmt.Sprintf("%d:1", c.calls), Model: "fixture", MaxNanodollars: 10_000_000})
	if err != nil {
		return nil, err
	}
	if err := c.repo.DispatchCreditAttempt(ctx, id); err != nil {
		return nil, err
	}
	if err := c.repo.SettleCreditAttempt(ctx, id, 1000); err != nil {
		return nil, err
	}
	if c.invalid {
		return []byte(`{"requirements":null}`), release()
	}
	return c.client.Interview(ctx, kind, raw, release)
}

func TestR3InterviewResultAndChargeAtomicity(t *testing.T) {
	for _, saveFailure := range []bool{false, true} {
		t.Run(fmt.Sprint("save_failure=", saveFailure), func(t *testing.T) {
			f := setup(t, 6)
			ctx := t.Context()
			repo := subRepo.NewSubscriptionRepository(f.pool)
			require.NoError(t, repo.ConfigureAIBudget(ctx, 100_000_000))
			client := &chargedInterviewFixtureClient{client: f.client, repo: repo, invalid: true}
			f.svc = NewService(f.store, client, subService.NewSubscriptionService(repo, nil, subService.BillingConfig{}))
			op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
			require.NoError(t, err)
			require.NoError(t, f.svc.RunOnce(ctx))
			b, err := repo.CreditBalance(ctx, f.owner)
			require.NoError(t, err)
			require.Equal(t, 30, b.Remaining)
			failed, err := scanOperation(f.pool.QueryRow(ctx, operationSelect+` AND id=$3`, f.owner, f.job, op.ID))
			require.NoError(t, err)
			require.True(t, failed.ChargeReleased)
			require.Equal(t, "AI_INVALID_RESPONSE", failed.ErrorCode)
			_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
			require.NoError(t, err)
			client.invalid = false
			if saveFailure {
				_, err = f.pool.Exec(ctx, `ALTER TABLE interview_versions ADD CONSTRAINT fixture_save_failure CHECK(false)`)
				require.NoError(t, err)
			}
			err = f.svc.RunOnce(ctx)
			if saveFailure {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			b, err = repo.CreditBalance(ctx, f.owner)
			require.NoError(t, err)
			if saveFailure {
				require.Equal(t, 30, b.Remaining)
			} else {
				require.Equal(t, 22, b.Remaining)
			}
			var versions int
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM interview_versions`).Scan(&versions))
			if saveFailure {
				require.Zero(t, versions)
			} else {
				require.Equal(t, 1, versions)
			}
			var cost int64
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&cost))
			require.EqualValues(t, 2000, cost)
		})
	}
}
