package ports

import (
	"context"

	"golang.org/x/oauth2"
)

// CalendarEvent represents a calendar event for the Google API
type CalendarEvent struct {
	Title       string
	Description string
	StartTime   string // RFC3339
	EndTime     string // RFC3339
}

// CreatedEvent represents the result of creating a calendar event
type CreatedEvent struct {
	EventID string
	Link    string
}

// GoogleCalendarClient defines the interface for Google Calendar API operations
type GoogleCalendarClient interface {
	CreateEvent(ctx context.Context, token *oauth2.Token, event *CalendarEvent) (*CreatedEvent, error)
	DeleteEvent(ctx context.Context, token *oauth2.Token, eventID string) error
	GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error)
}
