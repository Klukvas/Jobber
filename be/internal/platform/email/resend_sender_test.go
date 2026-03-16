package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// resendEmailRequest mirrors the Resend API request body.
type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func newTestResendSender(serverURL string) *ResendSender {
	sender := NewResendSender("test-api-key", "Jobber <noreply@example.com>")
	parsed, _ := url.Parse(serverURL + "/")
	sender.client.BaseURL = parsed
	return sender
}

func TestResendSender_SendVerificationEmail_Success(t *testing.T) {
	var received resendEmailRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "email-123"})
	}))
	defer server.Close()

	sender := newTestResendSender(server.URL)

	err := sender.SendVerificationEmail(context.Background(), "user@test.com", "verify-tok", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.From != "Jobber <noreply@example.com>" {
		t.Errorf("From = %q, want %q", received.From, "Jobber <noreply@example.com>")
	}
	if len(received.To) != 1 || received.To[0] != "user@test.com" {
		t.Errorf("To = %v, want [user@test.com]", received.To)
	}
	if received.Subject == "" {
		t.Error("Subject should not be empty")
	}
	if received.HTML == "" {
		t.Error("HTML should not be empty")
	}
}

func TestResendSender_SendPasswordResetEmail_Success(t *testing.T) {
	var received resendEmailRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "email-456"})
	}))
	defer server.Close()

	sender := newTestResendSender(server.URL)

	err := sender.SendPasswordResetEmail(context.Background(), "user@test.com", "reset-tok", "ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received.To) != 1 || received.To[0] != "user@test.com" {
		t.Errorf("To = %v, want [user@test.com]", received.To)
	}
	if received.Subject == "" {
		t.Error("Subject should not be empty")
	}
}

func TestResendSender_SendVerificationEmail_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
	}))
	defer server.Close()

	sender := newTestResendSender(server.URL)

	err := sender.SendVerificationEmail(context.Background(), "user@test.com", "tok", "en")
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}

func TestResendSender_SendPasswordResetEmail_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"message": "forbidden"})
	}))
	defer server.Close()

	sender := newTestResendSender(server.URL)

	err := sender.SendPasswordResetEmail(context.Background(), "user@test.com", "tok", "en")
	if err == nil {
		t.Fatal("expected error on API failure, got nil")
	}
}

func TestResendSender_UsesCorrectLocaleForTemplates(t *testing.T) {
	locales := []struct {
		locale     string
		wantSubstr string
	}{
		{"en", "Verify your email"},
		{"ru", "Подтверждение email"},
		{"ua", "Підтвердження email"},
	}

	for _, tt := range locales {
		t.Run(tt.locale, func(t *testing.T) {
			var received resendEmailRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&received)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"id": "id"})
			}))
			defer server.Close()

			sender := newTestResendSender(server.URL)
			sender.SendVerificationEmail(context.Background(), "a@b.com", "tok", tt.locale)

			if received.Subject != tt.wantSubstr+" — Jobber" {
				t.Errorf("Subject = %q, want %q", received.Subject, tt.wantSubstr+" — Jobber")
			}
		})
	}
}
