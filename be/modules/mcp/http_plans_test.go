package mcp

import (
	"context"
	"net/http"
	"testing"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subPorts "github.com/andreypavlenko/jobber/modules/subscriptions/ports"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type planRepository struct {
	subPorts.SubscriptionRepository
	plan string
}

func (r *planRepository) GetByUserID(_ context.Context, userID string) (*subModel.Subscription, error) {
	return &subModel.Subscription{UserID: userID, Plan: r.plan, Status: subService.StatusActive}, nil
}

func TestHTTPRealPaidPlanEnforcement(t *testing.T) {
	for _, plan := range []string{subService.PlanFree, subService.PlanPro, subService.PlanEnterprise} {
		t.Run(plan, func(t *testing.T) {
			repo := &planRepository{plan: plan}
			paid := subService.NewSubscriptionService(repo, nil, subService.BillingConfig{})
			jobs := newFakeJobs()
			tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}
			router := gin.New()
			registerHTTP(router, NewServer(jobs, nil, jobs, nil), tokens, paid, testHTTPConfig(), zap.NewNop())
			response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listCall)
			if plan == subService.PlanFree {
				require.Equal(t, 403, response.Code, response.Body.String())
				require.Zero(t, jobs.calls.Load())
			} else {
				require.Equal(t, 200, response.Code, response.Body.String())
				require.Contains(t, response.Body.String(), "Engineer")
			}
			// Downgrades take effect immediately even with an unchanged PAT.
			repo.plan = subService.PlanFree
			response = rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listCall)
			require.Equal(t, 403, response.Code, response.Body.String())
			require.Equal(t, int64(2), tokens.calls.Load())
		})
	}
}
