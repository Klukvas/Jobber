package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client sends messages to a Telegram chat via the Bot API.
type Client struct {
	botToken string
	chatID   string
	http     *http.Client
}

// NewClient creates a Telegram client. Returns nil if token or chatID is empty.
func NewClient(botToken, chatID string) *Client {
	if botToken == "" || chatID == "" {
		return nil
	}
	return &Client{
		botToken: botToken,
		chatID:   chatID,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends an HTML-formatted message to the configured chat.
func (c *Client) SendMessage(ctx context.Context, text string) error {
	payload := sendMessageRequest{
		ChatID:    c.chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.botToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: send request: %w", err)
	}
	defer resp.Body.Close()

	var tgResp telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("telegram: decode response: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram: API error: %s", tgResp.Description)
	}

	return nil
}
