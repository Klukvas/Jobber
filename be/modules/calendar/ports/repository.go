package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
)

// CalendarTokenRepository defines the interface for calendar token data access
type CalendarTokenRepository interface {
	Upsert(ctx context.Context, token *model.CalendarToken) error
	GetByUserID(ctx context.Context, userID string) (*model.CalendarToken, error)
	Delete(ctx context.Context, userID string) error
}

// CalendarStageRepository defines the interface for stage calendar_event_id operations
type CalendarStageRepository interface {
	SetCalendarEventID(ctx context.Context, stageID, eventID string) error
	ClearCalendarEventID(ctx context.Context, stageID string) error
	GetCalendarEventID(ctx context.Context, stageID string) (string, error)
	GetStageUserID(ctx context.Context, stageID string) (string, error)
}
