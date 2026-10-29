package http

import (
	"errors"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

// RespondAIError preserves the common allowance contract through module wrappers.
func RespondAIError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, model.ErrCreditUnavailable):
		RespondWithError(c, 403, "CREDIT_UNAVAILABLE", "Not enough AI credits for this operation")
	case errors.Is(err, model.ErrAIBudgetReached):
		RespondWithError(c, 503, "AI_BUDGET_REACHED", "AI spending protection stopped this operation. Credits may remain.")
	case errors.Is(err, model.ErrAIOperationConflict):
		RespondWithError(c, 409, "AI_OPERATION_CONFLICT", "This AI operation was already started or its input changed")
	default:
		return false
	}
	return true
}
