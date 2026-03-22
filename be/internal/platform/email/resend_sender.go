package email

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	"github.com/resend/resend-go/v2"
)

// ResendSender sends emails via the Resend API.
type ResendSender struct {
	client      *resend.Client
	fromAddress string
	breaker     *circuitbreaker.Breaker
}

// NewResendSender creates a new Resend email sender.
func NewResendSender(apiKey, fromAddress string) *ResendSender {
	return &ResendSender{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
		breaker:     circuitbreaker.New("resend", 3, 60*time.Second),
	}
}

func (s *ResendSender) SendVerificationEmail(_ context.Context, to, code, locale string) error {
	content := verificationEmail(code, locale)
	return s.breaker.Execute(func() error {
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
	})
}

func (s *ResendSender) SendPasswordResetEmail(_ context.Context, to, code, locale string) error {
	content := passwordResetEmail(code, locale)
	return s.breaker.Execute(func() error {
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
	})
}
