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

func (n *NoopSender) SendVerificationEmail(_ context.Context, to, code, _ string) error {
	n.log().Info("[DEV] verification email skipped — use this code to verify",
		zap.String("to", to),
		zap.String("code", code),
	)
	return nil
}

func (n *NoopSender) SendPasswordResetEmail(_ context.Context, to, code, _ string) error {
	n.log().Info("[DEV] password reset email skipped — use this code to reset",
		zap.String("to", to),
		zap.String("code", code),
	)
	return nil
}
