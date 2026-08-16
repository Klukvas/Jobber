package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReminderRepository struct {
	pool PgxDB
}

func NewReminderRepository(pool PgxDB) *ReminderRepository {
	return &ReminderRepository{pool: pool}
}

// Create inserts a reminder only if the target job belongs to the user (and, when
// a stage is given, that stage belongs to the job). The ownership check runs in
// the INSERT itself so a forged job_id can never attach a reminder to someone
// else's job. Zero rows affected means the job (or stage) was not found.
func (r *ReminderRepository) Create(ctx context.Context, reminder *model.Reminder) error {
	query := `
		INSERT INTO reminders (id, user_id, job_id, stage_id, remind_at, message, is_done, created_at, updated_at)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
		WHERE EXISTS (SELECT 1 FROM jobs WHERE id = $3 AND user_id = $2)
		  AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM job_stages WHERE id = $4 AND job_id = $3))
	`
	reminder.ID = uuid.New().String()
	now := time.Now().UTC()
	reminder.CreatedAt = now
	reminder.UpdatedAt = now

	result, err := r.pool.Exec(ctx, query,
		reminder.ID, reminder.UserID, reminder.JobID, reminder.StageID,
		reminder.RemindAt, reminder.Message, reminder.IsDone, reminder.CreatedAt, reminder.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}
	return nil
}

func (r *ReminderRepository) ListByUser(ctx context.Context, userID string) ([]*model.Reminder, error) {
	query := `
		SELECT id, user_id, job_id, stage_id, remind_at, message, is_done, created_at, updated_at
		FROM reminders WHERE user_id = $1 ORDER BY remind_at ASC
	`
	return r.queryReminders(ctx, query, userID)
}

func (r *ReminderRepository) ListByJob(ctx context.Context, userID, jobID string) ([]*model.Reminder, error) {
	query := `
		SELECT id, user_id, job_id, stage_id, remind_at, message, is_done, created_at, updated_at
		FROM reminders WHERE user_id = $1 AND job_id = $2 ORDER BY remind_at ASC
	`
	return r.queryReminders(ctx, query, userID, jobID)
}

func (r *ReminderRepository) queryReminders(ctx context.Context, query string, args ...any) ([]*model.Reminder, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reminders := make([]*model.Reminder, 0)
	for rows.Next() {
		rem := &model.Reminder{}
		if err := rows.Scan(&rem.ID, &rem.UserID, &rem.JobID, &rem.StageID, &rem.RemindAt, &rem.Message, &rem.IsDone, &rem.CreatedAt, &rem.UpdatedAt); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

// Update writes the mutable fields (message, remind_at, is_done) and is scoped by
// user_id so a reminder can only be changed by its owner. Zero rows affected
// means the reminder was not found for this user.
func (r *ReminderRepository) Update(ctx context.Context, reminder *model.Reminder) error {
	query := `
		UPDATE reminders
		SET message = $2, remind_at = $3, is_done = $4, updated_at = $5
		WHERE id = $1 AND user_id = $6
	`
	reminder.UpdatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query,
		reminder.ID, reminder.Message, reminder.RemindAt, reminder.IsDone, reminder.UpdatedAt, reminder.UserID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrReminderNotFound
	}
	return nil
}

func (r *ReminderRepository) Delete(ctx context.Context, userID, reminderID string) error {
	query := `DELETE FROM reminders WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, reminderID, userID)
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
		SELECT id, user_id, job_id, stage_id, remind_at, message, is_done, created_at, updated_at
		FROM reminders WHERE id = $1 AND user_id = $2
	`
	rem := &model.Reminder{}
	err := r.pool.QueryRow(ctx, query, reminderID, userID).Scan(&rem.ID, &rem.UserID, &rem.JobID, &rem.StageID, &rem.RemindAt, &rem.Message, &rem.IsDone, &rem.CreatedAt, &rem.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrReminderNotFound
		}
		return nil, err
	}
	return rem, nil
}
