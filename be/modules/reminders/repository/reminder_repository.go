package repository

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReminderRepository struct {
	pool *pgxpool.Pool
}

func NewReminderRepository(pool *pgxpool.Pool) *ReminderRepository {
	return &ReminderRepository{pool: pool}
}

func (r *ReminderRepository) Create(ctx context.Context, reminder *model.Reminder) error {
	query := `
		INSERT INTO reminders (id, user_id, application_id, stage_id, remind_at, message, is_done, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	reminder.ID = uuid.New().String()
	now := time.Now().UTC()
	reminder.CreatedAt = now
	reminder.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, reminder.ID, reminder.UserID, reminder.ApplicationID, reminder.StageID, reminder.RemindAt, reminder.Message, reminder.IsDone, reminder.CreatedAt, reminder.UpdatedAt)
	return err
}

func (r *ReminderRepository) ListByUser(ctx context.Context, userID string) ([]*model.Reminder, error) {
	query := `
		SELECT id, user_id, application_id, stage_id, remind_at, message, is_done, created_at, updated_at
		FROM reminders WHERE user_id = $1 ORDER BY remind_at ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reminders []*model.Reminder
	for rows.Next() {
		rem := &model.Reminder{}
		if err := rows.Scan(&rem.ID, &rem.UserID, &rem.ApplicationID, &rem.StageID, &rem.RemindAt, &rem.Message, &rem.IsDone, &rem.CreatedAt, &rem.UpdatedAt); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

func (r *ReminderRepository) Update(ctx context.Context, reminder *model.Reminder) error {
	query := `UPDATE reminders SET is_done = $2, updated_at = $3 WHERE id = $1`
	reminder.UpdatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, reminder.ID, reminder.IsDone, reminder.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrReminderNotFound
	}
	return nil
}

func (r *ReminderRepository) GetByID(ctx context.Context, userID, reminderID string) (*model.Reminder, error) {
	query := `
		SELECT id, user_id, application_id, stage_id, remind_at, message, is_done, created_at, updated_at
		FROM reminders WHERE id = $1 AND user_id = $2
	`
	rem := &model.Reminder{}
	err := r.pool.QueryRow(ctx, query, reminderID, userID).Scan(&rem.ID, &rem.UserID, &rem.ApplicationID, &rem.StageID, &rem.RemindAt, &rem.Message, &rem.IsDone, &rem.CreatedAt, &rem.UpdatedAt)
	if err != nil {
		return nil, model.ErrReminderNotFound
	}
	return rem, nil
}
