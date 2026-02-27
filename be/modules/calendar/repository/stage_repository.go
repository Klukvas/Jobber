package repository

import (
	"context"
	"errors"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StageRepository implements ports.CalendarStageRepository
type StageRepository struct {
	pool *pgxpool.Pool
}

// NewStageRepository creates a new stage repository for calendar operations
func NewStageRepository(pool *pgxpool.Pool) *StageRepository {
	return &StageRepository{pool: pool}
}

// SetCalendarEventID sets the calendar event ID on a stage
func (r *StageRepository) SetCalendarEventID(ctx context.Context, stageID, eventID string) error {
	query := `UPDATE application_stages SET calendar_event_id = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, stageID, eventID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageNotFound
	}
	return nil
}

// ClearCalendarEventID removes the calendar event ID from a stage
func (r *StageRepository) ClearCalendarEventID(ctx context.Context, stageID string) error {
	query := `UPDATE application_stages SET calendar_event_id = NULL WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, stageID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageNotFound
	}
	return nil
}

// GetCalendarEventID retrieves the calendar event ID for a stage
func (r *StageRepository) GetCalendarEventID(ctx context.Context, stageID string) (string, error) {
	query := `SELECT calendar_event_id FROM application_stages WHERE id = $1`
	var eventID *string
	err := r.pool.QueryRow(ctx, query, stageID).Scan(&eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", model.ErrStageNotFound
		}
		return "", err
	}
	if eventID == nil || *eventID == "" {
		return "", model.ErrEventNotFound
	}
	return *eventID, nil
}

// GetStageUserID retrieves the user ID who owns the stage
func (r *StageRepository) GetStageUserID(ctx context.Context, stageID string) (string, error) {
	query := `
		SELECT a.user_id
		FROM application_stages s
		JOIN applications a ON a.id = s.application_id
		WHERE s.id = $1
	`
	var userID string
	err := r.pool.QueryRow(ctx, query, stageID).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", model.ErrStageNotFound
		}
		return "", err
	}
	return userID, nil
}
