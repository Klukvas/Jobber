package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/andreypavlenko/jobber/modules/calendar/ports"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// GoogleClient implements ports.GoogleCalendarClient using the real Google API
type GoogleClient struct {
	oauthConfig *oauth2.Config
	breaker     *circuitbreaker.Breaker
}

// NewGoogleClient creates a new Google Calendar client
func NewGoogleClient(oauthConfig *oauth2.Config) *GoogleClient {
	return &GoogleClient{
		oauthConfig: oauthConfig,
		breaker:     circuitbreaker.New("google-calendar", 3, 30*time.Second),
	}
}

// CreateEvent creates a calendar event
func (c *GoogleClient) CreateEvent(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error) {
	srv, err := c.getService(ctx, token)
	if err != nil {
		return nil, err
	}

	gcalEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime,
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime,
		},
	}

	var created *calendar.Event
	err = c.breaker.Execute(func() error {
		var apiErr error
		created, apiErr = srv.Events.Insert("primary", gcalEvent).Do()
		return apiErr
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrCalendarAPI, err)
	}

	return &ports.CreatedEvent{
		EventID: created.Id,
		Link:    created.HtmlLink,
	}, nil
}

// DeleteEvent deletes a calendar event
func (c *GoogleClient) DeleteEvent(ctx context.Context, token *oauth2.Token, eventID string) error {
	srv, err := c.getService(ctx, token)
	if err != nil {
		return err
	}

	err = c.breaker.Execute(func() error {
		return srv.Events.Delete("primary", eventID).Do()
	})
	if err != nil {
		return fmt.Errorf("%w: %v", model.ErrCalendarAPI, err)
	}
	return nil
}

// GetUserEmail retrieves the email of the authenticated user
func (c *GoogleClient) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	srv, err := c.getService(ctx, token)
	if err != nil {
		return "", err
	}

	var calendarList *calendar.CalendarListEntry
	err = c.breaker.Execute(func() error {
		var apiErr error
		calendarList, apiErr = srv.CalendarList.Get("primary").Do()
		return apiErr
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", model.ErrCalendarAPI, err)
	}

	return calendarList.Id, nil
}

func (c *GoogleClient) getService(ctx context.Context, token *oauth2.Token) (*calendar.Service, error) {
	client := c.oauthConfig.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrCalendarAPI, err)
	}
	return srv, nil
}
