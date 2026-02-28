package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const maxPageTextLength = 50000

// ParsedJob represents structured job data extracted from a page.
type ParsedJob struct {
	Title       string  `json:"title"`
	CompanyName *string `json:"company_name,omitempty"`
	Source      *string `json:"source,omitempty"`
	URL         *string `json:"url,omitempty"`
	Description *string `json:"description,omitempty"`
}

// AnthropicClient wraps the Anthropic SDK for job parsing.
type AnthropicClient struct {
	client anthropic.Client
}

// NewAnthropicClient creates a new Anthropic API client.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicClient{client: client}
}

// ParseJobPage sends page text to Claude Haiku and extracts structured job data.
func (c *AnthropicClient) ParseJobPage(ctx context.Context, pageText, pageURL string) (*ParsedJob, error) {
	text := pageText
	if len(text) > maxPageTextLength {
		text = text[:maxPageTextLength]
	}

	systemPrompt := `You are a job posting parser. Extract structured data from the provided web page text.
Return ONLY valid JSON with these fields:
- "title" (string, required): the job title
- "company_name" (string or null): the company name
- "source" (string or null): the job board or website name (e.g. "LinkedIn", "Indeed", "DOU")
- "url" (string or null): the job posting URL (use the provided URL)
- "description" (string or null): a structured summary with key responsibilities, required technologies/skills, experience level, and salary if mentioned. Use bullet points with line breaks. Keep it concise but informative.

If you cannot determine a field, set it to null. Do not include any text outside the JSON object.`

	userMessage := fmt.Sprintf("Page URL: %s\n\nPage text:\n%s", pageURL, text)

	response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5_20251001,
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userMessage),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API call failed: %w", err)
	}

	if len(response.Content) == 0 {
		return nil, fmt.Errorf("empty response from anthropic")
	}

	var sb strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	responseText := strings.TrimSpace(sb.String())

	// Strip markdown code fences if present
	if strings.HasPrefix(responseText, "```") {
		lines := strings.Split(responseText, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
			responseText = strings.Join(lines, "\n")
		}
	}

	var parsed ParsedJob
	if err := json.Unmarshal([]byte(responseText), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	// Use provided URL as fallback
	if parsed.URL == nil {
		parsed.URL = &pageURL
	}

	return &parsed, nil
}
