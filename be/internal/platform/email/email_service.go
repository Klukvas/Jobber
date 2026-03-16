package email

import "context"

// Sender defines the email sending interface.
type Sender interface {
	SendVerificationEmail(ctx context.Context, to, code, locale string) error
	SendPasswordResetEmail(ctx context.Context, to, code, locale string) error
}
