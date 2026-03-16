package email

import (
	"context"

	"go.uber.org/zap"
)

// NoopSender is a no-op email sender used when Resend is not configured.
type NoopSender struct {
	Logger *zap.Logger
}

func (n *NoopSender) log() *zap.Logger {
	if n.Logger != nil {
		return n.Logger
	}
	return zap.NewNop()
}

func (n *NoopSender) SendVerificationEmail(_ context.Context, to, _, _ string) error {
	n.log().Debug("noop: verification email skipped", zap.String("to", to))
	return nil
}

func (n *NoopSender) SendPasswordResetEmail(_ context.Context, to, _, _ string) error {
	n.log().Debug("noop: password reset email skipped", zap.String("to", to))
	return nil
}
