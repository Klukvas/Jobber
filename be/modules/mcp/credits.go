package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	subHandler "github.com/andreypavlenko/jobber/modules/subscriptions/handler"
	"github.com/google/uuid"
)

var errAIUnavailable = errors.New("AI features are not available right now")
var errPDFUnavailable = errors.New("PDF export is not available")
var errAIProvider = errors.New("AI could not complete this request. Please try again later")

type AICredits interface {
	CheckLimit(context.Context, string, string) error
	FinishAIRequest(context.Context)
}

// Each invocation gets its own receipt, including calls sharing an MCP request.
func (t *tools) aiScope(ctx context.Context, owner, tool, kind string, input any) (context.Context, func(), error) {
	if t.ai.Credits == nil {
		return nil, nil, errAIUnavailable
	}
	canonical, err := json.Marshal(input)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(append([]byte(tool+":"), canonical...))
	scoped, finish := subHandler.BeginAIRequestScope(ctx, func() string { return owner }, uuid.NewString(), kind, hex.EncodeToString(sum[:]), t.ai.Credits)
	return scoped, finish, nil
}
