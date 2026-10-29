package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

type CreditStore interface {
	ReserveCreditAttempt(context.Context, subModel.AIAttempt) (string, error)
	DispatchCreditAttempt(context.Context, string) error
	SettleCreditAttempt(context.Context, string, int64) error
	ReleaseCreditAttempt(context.Context, string) error
	BlockAIProvider(context.Context, string) error
	AbortUnsentCreditAttempt(context.Context, string) error
}

var errProviderContract = errors.New("provider response violated the pricing contract")
var errProviderResponse = errors.New("provider failed after dispatch")
var ErrInterviewPreparationFailed = errors.New("interview source preparation failed before provider send")

type creditContextKey struct{}
type creditScope struct {
	mu          sync.Mutex
	owner       func() string
	key         string
	kind        string
	stage       int
	retry       string
	durable     bool
	fingerprint string
}

var OperationCosts = map[string]int{"tailor": 7, "cover_letter": 5, "match": 5, "resume_import": 7, "autofill_import": 7, "job_parse": 2, "bullet": 2, "summary": 2, "improve": 2, "ats": 3, "interview_plan": 8, "interview_feedback": 3, "interview_summary": 5}

// CreditContext establishes server-side identity and an operation receipt. The
// owner closure is evaluated only after authentication middleware has run.
func CreditContext(ctx context.Context, owner func() string, key, kind string) context.Context {
	if key == "" {
		key = uuid.NewString()
	}
	return context.WithValue(ctx, creditContextKey{}, &creditScope{owner: owner, key: key, kind: kind})
}
func InterviewCreditContext(ctx context.Context, owner, key, kind, retry string) context.Context {
	ctx = CreditContext(ctx, func() string { return owner }, key, "interview_"+kind)
	scope := ctx.Value(creditContextKey{}).(*creditScope)
	scope.retry = retry
	scope.durable = true
	return ctx
}
func CreditOperationFromContext(ctx context.Context, owner, receipt string) (subModel.CreditOperation, bool) {
	scope, ok := ctx.Value(creditContextKey{}).(*creditScope)
	if !ok {
		return subModel.CreditOperation{}, false
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if receipt != "" {
		scope.key = receipt
	}
	if scope.owner == nil || scope.owner() != owner {
		return subModel.CreditOperation{}, false
	}
	cost, known := OperationCosts[scope.kind]
	if !known {
		return subModel.CreditOperation{}, false
	}
	hash := sha256.Sum256([]byte(scope.kind + ":" + scope.key))
	fingerprint := scope.fingerprint
	if fingerprint == "" {
		fingerprint = hex.EncodeToString(hash[:])
	}
	return subModel.CreditOperation{UserID: owner, Key: scope.key, Kind: scope.kind, Fingerprint: fingerprint, Credits: cost}, true
}
func CurrentCreditOperation(ctx context.Context) (subModel.CreditOperation, bool, bool) {
	scope, ok := ctx.Value(creditContextKey{}).(*creditScope)
	if !ok || scope.owner == nil {
		return subModel.CreditOperation{}, false, false
	}
	op, found := CreditOperationFromContext(ctx, scope.owner(), "")
	return op, scope.durable, found
}
func SetDurableCreditContext(ctx context.Context) {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		scope.durable = true
		scope.mu.Unlock()
	}
}

func (c *AnthropicClient) SetCreditStore(store CreditStore) { c.creditStore = store }

// Published context windows provide a hard upper bound independent of tokenizer
// estimates or language. No prompt cache, tools, batch discounts or other models
// are admitted. Output includes reasoning; max_tokens is its server-side cap.
func maximumProviderLiability(params anthropic.MessageNewParams) (int64, error) {
	if params.MaxTokens <= 0 || params.MaxTokens > 12000 || len(params.Tools) > 0 {
		return 0, subModel.ErrAIBudgetReached
	}
	encoded, err := params.MarshalJSON()
	if err != nil {
		return 0, err
	}
	// Cache creation premiums cannot be bounded at the ordinary input rate.
	if containsBudgetUnsupported(encoded) {
		return 0, subModel.ErrAIBudgetReached
	}
	switch params.Model {
	case anthropic.Model("claude-sonnet-5-5"):
		return 2_000_000_000 + params.MaxTokens*10_000, nil
	case anthropic.ModelClaudeHaiku4_5_20251001:
		return 200_000_000 + params.MaxTokens*5_000, nil
	default:
		return 0, subModel.ErrAIBudgetReached
	}
}
func providerActualCost(params anthropic.MessageNewParams, response *anthropic.Message, reserved int64) (int64, error) {
	if response == nil {
		return 0, subModel.ErrAIBudgetReached
	}
	breach := errors.Join(subModel.ErrAIBudgetReached, errProviderContract)
	var inputRate, outputRate, maxInput int64
	switch params.Model {
	case anthropic.Model("claude-sonnet-5-5"):
		inputRate, outputRate, maxInput = 2000, 10000, 1_000_000
	case anthropic.ModelClaudeHaiku4_5_20251001:
		inputRate, outputRate, maxInput = 1000, 5000, 200_000
	default:
		return 0, breach
	}
	var usageFields map[string]json.RawMessage
	_ = json.Unmarshal([]byte(response.Usage.RawJSON()), &usageFields)
	input, inputValid := billingTokenCount(usageFields["input_tokens"])
	output, outputValid := billingTokenCount(usageFields["output_tokens"])
	malformedBilling, premiumBilling := optionalBillingUsage(usageFields)
	// Missing metadata cannot hide a separately present contract violation.
	if (response.JSON.Model.Valid() && response.Model != params.Model) ||
		(inputValid && (input <= 0 || input > maxInput)) ||
		(outputValid && (output < 0 || output > params.MaxTokens)) ||
		premiumBilling {
		return 0, breach
	}
	if !response.JSON.Model.Valid() || !response.JSON.Usage.Valid() || !inputValid || !outputValid || malformedBilling {
		return 0, subModel.ErrAIBudgetReached
	}
	actual := input*inputRate + output*outputRate
	if actual > reserved {
		return 0, breach
	}
	return actual, nil
}

func billingTokenCount(raw json.RawMessage) (int64, bool) {
	var value *int64
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return 0, false
	}
	return *value, true
}

// Optional billing metadata may be omitted, but present values must have strict
// JSON types. The SDK also coerces strings/counters and non-object containers.
func optionalBillingUsage(fields map[string]json.RawMessage) (malformed, premium bool) {
	if fields == nil {
		return true, false
	}
	count := func(fields map[string]json.RawMessage, key string) {
		raw, present := fields[key]
		if !present {
			return
		}
		value, valid := billingTokenCount(raw)
		if !valid || value < 0 {
			malformed = true
		} else if value > 0 {
			premium = true
		}
	}
	count(fields, "cache_creation_input_tokens")
	count(fields, "cache_read_input_tokens")
	for container, keys := range map[string][]string{
		"cache_creation":  {"ephemeral_1h_input_tokens", "ephemeral_5m_input_tokens"},
		"server_tool_use": {"web_search_requests", "web_fetch_requests"},
	} {
		if raw, present := fields[container]; present {
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(raw, &nested); err != nil || nested == nil {
				malformed = true
				continue
			}
			for _, key := range keys {
				count(nested, key)
			}
		}
	}
	if raw, present := fields["service_tier"]; present {
		var tier *string
		if err := json.Unmarshal(raw, &tier); err != nil || tier == nil {
			malformed = true
		} else {
			switch *tier {
			case "standard", "batch":
			case "priority":
				premium = true
			default:
				malformed = true
			}
		}
	}
	return malformed, premium
}

func (c *AnthropicClient) budgetedMessage(ctx context.Context, params anthropic.MessageNewParams, invoke func() (*anthropic.Message, error)) (*anthropic.Message, error) {
	return c.budgetedPreparedMessage(ctx, params, nil, func(func() error) (*anthropic.Message, error) { return invoke() })
}
func (c *AnthropicClient) budgetedPreparedMessage(ctx context.Context, params anthropic.MessageNewParams, prepare func() (func() error, error), invoke func(func() error) (*anthropic.Message, error)) (*anthropic.Message, error) {
	var response *anthropic.Message
	run := func() error {
		var err error
		response, err = c.admitBudgetedMessage(ctx, params, prepare, invoke)
		return err
	}
	if c.breaker == nil {
		err := run()
		return response, err
	}
	err := c.breaker.ExecuteIgnoring(run, func(err error) bool {
		return !errors.Is(err, errProviderResponse) && (errors.Is(err, subModel.ErrCreditUnavailable) || errors.Is(err, subModel.ErrAIBudgetReached) || errors.Is(err, subModel.ErrAIOperationConflict))
	})
	return response, err
}
func (c *AnthropicClient) admitBudgetedMessage(ctx context.Context, params anthropic.MessageNewParams, prepare func() (func() error, error), invoke func(func() error) (*anthropic.Message, error)) (*anthropic.Message, error) {
	// Unit-test clients are deliberately built with a fake transport and no store.
	// Production construction requires the durable gate even if a caller omits context.
	if c.creditStore == nil {
		if c.requireCreditStore && c.callAPIFunc == nil {
			return nil, subModel.ErrAIBudgetReached
		}
		release := func() error { return nil }
		if prepare != nil {
			var err error
			release, err = prepare()
			if err != nil {
				return nil, errors.Join(ErrInterviewPreparationFailed, err)
			}
		}
		response, err := invoke(release)
		return response, errors.Join(err, release())
	}
	scope, ok := ctx.Value(creditContextKey{}).(*creditScope)
	if !ok {
		return nil, subModel.ErrCreditUnavailable
	}
	op, _, ok := CurrentCreditOperation(ctx)
	if !ok {
		return nil, subModel.ErrCreditUnavailable
	}
	scope.mu.Lock()
	scope.stage++
	stage := scope.stage
	retry := scope.retry
	scope.mu.Unlock()
	maxStages := 1
	if op.Kind == "tailor" || op.Kind == "cover_letter" {
		maxStages = 2
	}
	if stage > maxStages {
		return nil, subModel.ErrAIOperationConflict
	}
	if err := validateOperationStage(op.Kind, stage, params); err != nil {
		return nil, err
	}
	reserve, err := maximumProviderLiability(params)
	if err != nil {
		return nil, err
	}
	id, err := c.creditStore.ReserveCreditAttempt(ctx, subModel.AIAttempt{Operation: op, Stage: fmt.Sprintf("%s:%d", retry, stage), Model: string(params.Model), MaxNanodollars: reserve})
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, errors.Join(err, c.creditStore.ReleaseCreditAttempt(context.WithoutCancel(ctx), id))
	}
	if err = c.creditStore.DispatchCreditAttempt(ctx, id); err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		return nil, errors.Join(err, c.creditStore.ReleaseCreditAttempt(cleanup, id))
	}
	release := func() error { return nil }
	if prepare != nil {
		var prepareErr error
		release, prepareErr = prepare()
		if prepareErr != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancel()
			return nil, errors.Join(ErrInterviewPreparationFailed, prepareErr, c.creditStore.AbortUnsentCreditAttempt(cleanup, id))
		}
	}
	if err := ctx.Err(); err != nil {
		releaseErr := release()
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		return nil, errors.Join(err, releaseErr, c.creditStore.AbortUnsentCreditAttempt(cleanup, id))
	}
	response, callErr := invoke(release)
	// The source transaction must release its sole pool connection BEFORE any
	// monetary settlement, block persistence or compensation needs PostgreSQL.
	callErr = errors.Join(callErr, release())
	if response != nil {
		actual, usageErr := providerActualCost(params, response, reserve)
		if usageErr != nil {
			if errors.Is(usageErr, errProviderContract) {
				block, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
				blockErr := c.creditStore.BlockAIProvider(block, "response_contract")
				cancel()
				return nil, errors.Join(errProviderResponse, callErr, usageErr, blockErr)
			}
			return nil, errors.Join(errProviderResponse, callErr, usageErr)
		}
		settle, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if err := c.creditStore.SettleCreditAttempt(settle, id, actual); err != nil {
			return nil, errors.Join(errProviderResponse, callErr, err)
		}
	}
	if callErr != nil {
		return response, errors.Join(errProviderResponse, callErr)
	}
	if response == nil {
		return nil, errors.Join(errProviderResponse, subModel.ErrAIBudgetReached)
	}
	return response, nil
}

func containsBudgetUnsupported(body []byte) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return true
	}
	allowed := map[string]bool{"model": true, "max_tokens": true, "messages": true, "system": true, "thinking": true, "output_config": true}
	for key := range fields {
		if !allowed[key] {
			return true
		}
	}
	return bytes.Contains(body, []byte(`"cache_control"`)) || bytes.Contains(body, []byte(`"speed"`)) || bytes.Contains(body, []byte(`"inference_geo"`))
}

func SetCreditFingerprint(ctx context.Context, fingerprint string) {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		scope.fingerprint = fingerprint
		scope.mu.Unlock()
	}
}

func validateOperationStage(kind string, stage int, params anthropic.MessageNewParams) error {
	caps := map[string]int64{"job_parse": 2048, "match": 4096, "bullet": 1024, "summary": 512, "improve": 1024, "ats": 2048, "resume_import": 4096, "autofill_import": 4096, "interview_plan": 12000, "interview_feedback": 12000, "interview_summary": 12000}
	if kind == "tailor" || kind == "cover_letter" {
		cap := int64(1500)
		if stage == 2 {
			cap = 512
			if kind == "cover_letter" {
				cap = 768
			}
		}
		if params.Model != "claude-sonnet-5-5" || params.MaxTokens != cap {
			return subModel.ErrAIBudgetReached
		}
		return nil
	}
	cap, ok := caps[kind]
	if !ok || params.Model != anthropic.ModelClaudeHaiku4_5_20251001 || params.MaxTokens > cap {
		return subModel.ErrAIBudgetReached
	}
	return nil
}
