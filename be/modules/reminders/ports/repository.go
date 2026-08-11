package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
)

// ReminderRepository defines the data-access contract for reminders.
type ReminderRepository interface {
	Create(ctx context.Context, reminder *model.Reminder) error
	ListByUser(ctx context.Context, userID string) ([]*model.Reminder, error)
	ListByJob(ctx context.Context, userID, jobID string) ([]*model.Reminder, error)
	GetByID(ctx context.Context, userID, reminderID string) (*model.Reminder, error)
	Update(ctx context.Context, reminder *model.Reminder) error
	Delete(ctx context.Context, userID, reminderID string) error
}
