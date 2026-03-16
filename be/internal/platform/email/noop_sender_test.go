package email

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNoopSender_SendVerificationEmail(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	sender := &NoopSender{Logger: zap.New(core)}

	err := sender.SendVerificationEmail(context.Background(), "user@example.com", "token123", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Message != "noop: verification email skipped" {
		t.Errorf("unexpected log message: %q", entry.Message)
	}

	toField := entry.ContextMap()["to"]
	if toField != "user@example.com" {
		t.Errorf("expected to=user@example.com, got %v", toField)
	}
}

func TestNoopSender_SendPasswordResetEmail(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	sender := &NoopSender{Logger: zap.New(core)}

	err := sender.SendPasswordResetEmail(context.Background(), "user@example.com", "reset-tok", "ru")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	entry := logs.All()[0]
	if entry.Message != "noop: password reset email skipped" {
		t.Errorf("unexpected log message: %q", entry.Message)
	}
}

func TestNoopSender_NilLogger(t *testing.T) {
	sender := &NoopSender{}

	// Should not panic with nil logger
	err := sender.SendVerificationEmail(context.Background(), "a@b.com", "t", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = sender.SendPasswordResetEmail(context.Background(), "a@b.com", "t", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
