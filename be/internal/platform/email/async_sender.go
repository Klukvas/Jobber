package email

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

const defaultMaxConcurrent = 10

// AsyncSender wraps any Sender and dispatches emails in background goroutines.
// Errors are logged instead of returned to the caller.
type AsyncSender struct {
	inner  Sender
	logger *zap.Logger
	sem    chan struct{} // semaphore to limit concurrent sends
}

// NewAsyncSender creates an AsyncSender that delegates to inner.
// maxConcurrent controls how many email sends can run in parallel.
// If maxConcurrent <= 0, defaultMaxConcurrent (10) is used.
func NewAsyncSender(inner Sender, logger *zap.Logger, maxConcurrent int) *AsyncSender {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrent
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AsyncSender{
		inner:  inner,
		logger: logger,
		sem:    make(chan struct{}, maxConcurrent),
	}
}

// SendVerificationEmail enqueues the verification email to be sent asynchronously.
// It always returns nil; delivery errors are logged.
func (a *AsyncSender) SendVerificationEmail(_ context.Context, to, code, locale string) error {
	select {
	case a.sem <- struct{}{}:
		go func() {
			defer func() { <-a.sem }()
			ctx := context.Background()
			if err := a.inner.SendVerificationEmail(ctx, to, code, locale); err != nil {
				a.logger.Error("async: failed to send verification email",
					zap.String("to", maskEmail(to)),
					zap.Error(err),
				)
			}
		}()
	default:
		a.logger.Warn("async: semaphore full, dropping verification email",
			zap.String("to", maskEmail(to)),
		)
	}
	return nil
}

// SendPasswordResetEmail enqueues the password-reset email to be sent asynchronously.
// It always returns nil; delivery errors are logged.
func (a *AsyncSender) SendPasswordResetEmail(_ context.Context, to, code, locale string) error {
	select {
	case a.sem <- struct{}{}:
		go func() {
			defer func() { <-a.sem }()
			ctx := context.Background()
			if err := a.inner.SendPasswordResetEmail(ctx, to, code, locale); err != nil {
				a.logger.Error("async: failed to send password reset email",
					zap.String("to", maskEmail(to)),
					zap.Error(err),
				)
			}
		}()
	default:
		a.logger.Warn("async: semaphore full, dropping password reset email",
			zap.String("to", maskEmail(to)),
		)
	}
	return nil
}

// maskEmail masks the local part of an email address for safe logging.
// "john@example.com" → "j***@example.com"
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	return string(email[0]) + "***" + email[at:]
}
