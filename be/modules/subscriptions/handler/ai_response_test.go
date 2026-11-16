package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type resultOutcomeLedger struct {
	*MockSubscriptionRepository
	complete func() error
	rejected int
}

func (l *resultOutcomeLedger) CompleteCreditOperation(context.Context, model.CreditOperation) error {
	return l.complete()
}
func (l *resultOutcomeLedger) RejectCreditOperation(context.Context, model.CreditOperation) error {
	l.rejected++
	return nil
}

func TestAIResponseWaitsForChargeCommit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "save failure"}[fail], func(t *testing.T) {
			w := httptest.NewRecorder()
			commits := 0
			ledger := &resultOutcomeLedger{MockSubscriptionRepository: &MockSubscriptionRepository{}, complete: func() error {
				commits++
				if commits == 1 {
					require.Empty(t, w.Body.String(), "result must not be sent before charge commits")
				}
				if fail {
					return errors.New("synthetic database failure")
				}
				return nil
			}}
			svc := subService.NewSubscriptionService(ledger, nil, subService.BillingConfig{})
			r := gin.New()
			r.Use(AIRequestScope(svc))
			r.POST("/api/v1/cover-letters/ai/generate", func(c *gin.Context) {
				c.Set("user_id", "owner")
				ai.MarkCreditAdmission(c.Request.Context())
				c.JSON(200, gin.H{"result": "usable text"})
			})
			r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/cover-letters/ai/generate", strings.NewReader(`{}`)))
			if fail {
				require.Equal(t, 503, w.Code)
				require.NotContains(t, w.Body.String(), "usable text")
				require.Contains(t, w.Body.String(), "AI_RESULT_SAVE_FAILED")
				require.Equal(t, 1, ledger.rejected)
			} else {
				require.Equal(t, 200, w.Code)
				require.Contains(t, w.Body.String(), "usable text")
				require.Zero(t, ledger.rejected)
			}
		})
	}
}

func TestAIRejectedDuplicateRequestCannotRefundAdmittedOperation(t *testing.T) {
	ledger := &resultOutcomeLedger{MockSubscriptionRepository: &MockSubscriptionRepository{}, complete: func() error { t.Fatal("conflicting request cannot complete another charge"); return nil }}
	svc := subService.NewSubscriptionService(ledger, nil, subService.BillingConfig{})
	r := gin.New()
	r.Use(AIRequestScope(svc))
	r.POST("/api/v1/cover-letters/ai/generate", func(c *gin.Context) {
		c.Set("user_id", "owner")
		c.JSON(409, gin.H{"error_code": "AI_OPERATION_CONFLICT"})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/cover-letters/ai/generate", strings.NewReader(`{}`)))
	require.Equal(t, 409, w.Code)
	require.Zero(t, ledger.rejected)
}

func TestCancelledAIResponseCannotPublishSuccessOrCharge(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ledger := &resultOutcomeLedger{MockSubscriptionRepository: &MockSubscriptionRepository{}, complete: func() error { t.Fatal("cancelled response cannot charge"); return nil }}
	svc := subService.NewSubscriptionService(ledger, nil, subService.BillingConfig{})
	r := gin.New()
	r.Use(AIRequestScope(svc))
	r.POST("/api/v1/cover-letters/ai/generate", func(c *gin.Context) {
		c.Set("user_id", "owner")
		ai.MarkCreditAdmission(c.Request.Context())
		ai.MarkCreditResult(c.Request.Context())
		cancel()
		c.JSON(200, gin.H{"result": "usable text"})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/cover-letters/ai/generate", strings.NewReader(`{}`)).WithContext(ctx))
	require.Equal(t, 503, w.Code)
	require.NotContains(t, w.Body.String(), "usable text")
	require.Equal(t, 1, ledger.rejected)
}

func TestCompleteAIRequestCancellationSharedWithMCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ai.CreditContext(ctx, func() string { return "owner" }, "key", "match")
	ai.MarkCreditAdmission(ctx)
	ai.MarkCreditResult(ctx)
	ledger := &resultOutcomeLedger{MockSubscriptionRepository: &MockSubscriptionRepository{}, complete: func() error { t.Fatal("cancelled MCP response must not charge"); return nil }}
	svc := subService.NewSubscriptionService(ledger, nil, subService.BillingConfig{})
	cancel()
	require.ErrorIs(t, svc.CompleteAIRequest(ctx), context.Canceled)
	svc.FinishAIRequest(ctx)
	require.Equal(t, 1, ledger.rejected)
	require.False(t, ai.CreditResultReady(ctx))
}
