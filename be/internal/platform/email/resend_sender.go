package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	client      *resend.Client
	fromAddress string
}

// NewResendSender creates a new Resend email sender.
func NewResendSender(apiKey, fromAddress string) *ResendSender {
	return &ResendSender{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
	}
}

func (s *ResendSender) SendVerificationEmail(_ context.Context, to, code, locale string) error {
	content := verificationEmail(code, locale)
	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: content.Subject,
		Html:    content.HTML,
	})
	if err != nil {
		return fmt.Errorf("resend: send verification email: %w", err)
	}
	return nil
}

func (s *ResendSender) SendPasswordResetEmail(_ context.Context, to, code, locale string) error {
	content := passwordResetEmail(code, locale)
	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.fromAddress,
		To:      []string{to},
		Subject: content.Subject,
		Html:    content.HTML,
	})
	if err != nil {
		return fmt.Errorf("resend: send password reset email: %w", err)
	}
	return nil
}
