package ai

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

// NewTestClient creates an AnthropicClient with a custom callAPIFunc for testing.
// This allows external packages to mock the AI API call without hitting real endpoints.
func NewTestClient(fn func(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error)) *AnthropicClient {
	return &AnthropicClient{
		callAPIFunc: fn,
	}
}
