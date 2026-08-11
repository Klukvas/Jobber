package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockReminderRepository implements ports.ReminderRepository.
type MockReminderRepository struct {
	CreateFunc     func(ctx context.Context, r *model.Reminder) error
	ListByUserFunc func(ctx context.Context, userID string) ([]*model.Reminder, error)
	ListByJobFunc  func(ctx context.Context, userID, jobID string) ([]*model.Reminder, error)
	GetByIDFunc    func(ctx context.Context, userID, reminderID string) (*model.Reminder, error)
	UpdateFunc     func(ctx context.Context, r *model.Reminder) error
	DeleteFunc     func(ctx context.Context, userID, reminderID string) error
}

func (m *MockReminderRepository) Create(ctx context.Context, r *model.Reminder) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, r)
	}
	return nil
}

func (m *MockReminderRepository) ListByUser(ctx context.Context, userID string) ([]*model.Reminder, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockReminderRepository) ListByJob(ctx context.Context, userID, jobID string) ([]*model.Reminder, error) {
	if m.ListByJobFunc != nil {
		return m.ListByJobFunc(ctx, userID, jobID)
	}
	return nil, nil
}

func (m *MockReminderRepository) GetByID(ctx context.Context, userID, reminderID string) (*model.Reminder, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, reminderID)
	}
	return nil, nil
}

func (m *MockReminderRepository) Update(ctx context.Context, r *model.Reminder) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, r)
	}
	return nil
}

func (m *MockReminderRepository) Delete(ctx context.Context, userID, reminderID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, reminderID)
	}
	return nil
}

func TestReminderService_Create(t *testing.T) {
	userID := "user-123"
	remindAt := time.Now().Add(24 * time.Hour)

	t.Run("creates reminder and trims the message", func(t *testing.T) {
		var captured *model.Reminder
		repo := &MockReminderRepository{
			CreateFunc: func(_ context.Context, r *model.Reminder) error {
				r.ID = "rem-1"
				captured = r
				return nil
			},
		}
		svc := NewReminderService(repo)

		dto, err := svc.Create(context.Background(), userID, &model.CreateReminderRequest{
			JobID:    "job-1",
			RemindAt: remindAt,
			Message:  "  call the recruiter  ",
		})

		require.NoError(t, err)
		assert.Equal(t, "rem-1", dto.ID)
		assert.Equal(t, "call the recruiter", dto.Message)
		assert.False(t, dto.IsDone)
		assert.Equal(t, userID, captured.UserID)
	})

	t.Run("rejects an empty message before hitting the repo", func(t *testing.T) {
		called := false
		repo := &MockReminderRepository{
			CreateFunc: func(_ context.Context, _ *model.Reminder) error {
				called = true
				return nil
			},
		}
		svc := NewReminderService(repo)

		_, err := svc.Create(context.Background(), userID, &model.CreateReminderRequest{
			JobID:    "job-1",
			RemindAt: remindAt,
			Message:  "   ",
		})

		assert.ErrorIs(t, err, model.ErrMessageRequired)
		assert.False(t, called, "repo.Create must not be called on invalid input")
	})

	t.Run("propagates job-not-found from the repo (ownership check)", func(t *testing.T) {
		repo := &MockReminderRepository{
			CreateFunc: func(_ context.Context, _ *model.Reminder) error {
				return model.ErrJobNotFound
			},
		}
		svc := NewReminderService(repo)

		_, err := svc.Create(context.Background(), userID, &model.CreateReminderRequest{
			JobID:    "not-mine",
			RemindAt: remindAt,
			Message:  "hi",
		})

		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})
}

func TestReminderService_Update(t *testing.T) {
	userID := "user-123"
	base := func() *model.Reminder {
		return &model.Reminder{
			ID:       "rem-1",
			UserID:   userID,
			JobID:    "job-1",
			Message:  "old message",
			RemindAt: time.Now(),
			IsDone:   false,
		}
	}

	t.Run("toggles is_done and leaves other fields untouched", func(t *testing.T) {
		var saved *model.Reminder
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Reminder, error) {
				return base(), nil
			},
			UpdateFunc: func(_ context.Context, r *model.Reminder) error {
				saved = r
				return nil
			},
		}
		svc := NewReminderService(repo)

		done := true
		dto, err := svc.Update(context.Background(), userID, "rem-1", &model.UpdateReminderRequest{IsDone: &done})

		require.NoError(t, err)
		assert.True(t, dto.IsDone)
		assert.Equal(t, "old message", dto.Message, "message must be unchanged when not provided")
		assert.True(t, saved.IsDone)
	})

	t.Run("updates the message and trims it", func(t *testing.T) {
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Reminder, error) {
				return base(), nil
			},
		}
		svc := NewReminderService(repo)

		newMsg := "  new note  "
		dto, err := svc.Update(context.Background(), userID, "rem-1", &model.UpdateReminderRequest{Message: &newMsg})

		require.NoError(t, err)
		assert.Equal(t, "new note", dto.Message)
	})

	t.Run("rejects an empty message", func(t *testing.T) {
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Reminder, error) {
				return base(), nil
			},
		}
		svc := NewReminderService(repo)

		empty := "  "
		_, err := svc.Update(context.Background(), userID, "rem-1", &model.UpdateReminderRequest{Message: &empty})

		assert.ErrorIs(t, err, model.ErrMessageRequired)
	})

	t.Run("propagates not-found from GetByID (ownership check)", func(t *testing.T) {
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Reminder, error) {
				return nil, model.ErrReminderNotFound
			},
		}
		svc := NewReminderService(repo)

		done := true
		_, err := svc.Update(context.Background(), userID, "missing", &model.UpdateReminderRequest{IsDone: &done})

		assert.ErrorIs(t, err, model.ErrReminderNotFound)
	})
}

func TestReminderService_ListAndDelete(t *testing.T) {
	userID := "user-123"

	t.Run("ListByUser maps to DTOs", func(t *testing.T) {
		repo := &MockReminderRepository{
			ListByUserFunc: func(_ context.Context, _ string) ([]*model.Reminder, error) {
				return []*model.Reminder{{ID: "a"}, {ID: "b"}}, nil
			},
		}
		svc := NewReminderService(repo)

		dtos, err := svc.ListByUser(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, dtos, 2)
	})

	t.Run("Delete delegates to the repo", func(t *testing.T) {
		var gotUser, gotID string
		repo := &MockReminderRepository{
			DeleteFunc: func(_ context.Context, u, id string) error {
				gotUser, gotID = u, id
				return nil
			},
		}
		svc := NewReminderService(repo)

		err := svc.Delete(context.Background(), userID, "rem-9")

		require.NoError(t, err)
		assert.Equal(t, userID, gotUser)
		assert.Equal(t, "rem-9", gotID)
	})
}
