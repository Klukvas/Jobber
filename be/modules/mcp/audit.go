package mcp

import (
	"context"
	"time"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const unknownAuditTool = "unknown_tool"

// AuditEntry deliberately has no fields for arguments, output, or error text.
type AuditEntry struct {
	ID         string
	UserID     string
	APITokenID *string
	Tool       string
	OK         bool
	ErrorCode  *string
	DurationMS int
}

type AuditRecorder interface {
	Record(context.Context, AuditEntry) error
}

func WithAuditWriter(writer *AuditWriter) ServerOption {
	return func(t *tools) { t.audit = writer }
}

type auditOutcomeKey struct{}

type auditOutcome struct {
	code string
}

// The SDK validates inputs before the typed register handler and serializes
// outputs afterwards. Observe that whole boundary, retaining only a stable code.
func (t *tools) auditMiddleware(next sdk.MethodHandler) sdk.MethodHandler {
	return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
		call, ok := req.(*sdk.CallToolRequest)
		if method != "tools/call" || !ok || call.Params == nil {
			return next(ctx, method, req)
		}
		tool := unknownAuditTool
		registered := t.auditTools[call.Params.Name]
		if registered {
			tool = call.Params.Name
		}
		started := time.Now()
		outcome := &auditOutcome{code: "validation"}
		ctx, finishSpan := startAuditSpan(ctx, t.auditTrace, tool)
		result, err := next(context.WithValue(ctx, auditOutcomeKey{}, outcome), method, req)
		if err != nil {
			outcome.code = "internal"
		}
		if !registered {
			outcome.code = "unknown_tool"
		}
		finishSpan(outcome.code)
		if t.audit != nil && call.Extra != nil && call.Extra.TokenInfo != nil && call.Extra.TokenInfo.UserID != "" {
			entry := AuditEntry{
				ID: uuid.NewString(), UserID: call.Extra.TokenInfo.UserID,
				Tool: tool, OK: outcome.code == "", DurationMS: int(time.Since(started).Milliseconds()),
			}
			if tokenID, ok := call.Extra.TokenInfo.Extra["api_token_id"].(string); ok && tokenID != "" {
				entry.APITokenID = &tokenID
			}
			if outcome.code != "" {
				code := outcome.code
				entry.ErrorCode = &code
			}
			t.audit.Enqueue(entry)
		}
		return result, err
	}
}
