package service

import (
	"context"
	"fmt"
	"html"

	"github.com/andreypavlenko/jobber/internal/platform/telegram"
	userRepo "github.com/andreypavlenko/jobber/modules/users/repository"
)

// SupportService handles support ticket submission via Telegram.
type SupportService struct {
	tg       *telegram.Client
	userRepo *userRepo.UserRepository
}

// NewSupportService creates a new support service.
func NewSupportService(tg *telegram.Client, userRepo *userRepo.UserRepository) *SupportService {
	return &SupportService{tg: tg, userRepo: userRepo}
}

// Submit looks up the user by ID, formats and sends a support message to Telegram.
func (s *SupportService) Submit(ctx context.Context, userID, subject, message, page string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("support: find user: %w", err)
	}

	pageInfo := ""
	if page != "" {
		pageInfo = fmt.Sprintf("<b>Page:</b> %s\n", html.EscapeString(page))
	}

	text := fmt.Sprintf(
		"<b>📩 Support Request — Jobber</b>\n\n"+
			"<b>From:</b> %s (%s)\n"+
			"%s"+
			"<b>Subject:</b> %s\n\n"+
			"<b>Message:</b>\n%s",
		html.EscapeString(user.Name),
		html.EscapeString(user.Email),
		pageInfo,
		html.EscapeString(subject),
		html.EscapeString(message),
	)

	if err := s.tg.SendMessage(ctx, text); err != nil {
		return fmt.Errorf("support: send telegram message: %w", err)
	}

	return nil
}
