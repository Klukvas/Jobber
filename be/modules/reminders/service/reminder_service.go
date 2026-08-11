package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/andreypavlenko/jobber/modules/reminders/ports"
)

type ReminderService struct {
	repo ports.ReminderRepository
}

func NewReminderService(repo ports.ReminderRepository) *ReminderService {
	return &ReminderService{repo: repo}
}

func (s *ReminderService) Create(ctx context.Context, userID string, req *model.CreateReminderRequest) (*model.ReminderDTO, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, model.ErrMessageRequired
	}

	reminder := &model.Reminder{
		UserID:   userID,
		JobID:    req.JobID,
		StageID:  req.StageID,
		RemindAt: req.RemindAt,
		Message:  message,
		IsDone:   false,
	}

	if err := s.repo.Create(ctx, reminder); err != nil {
		return nil, err
	}
	return reminder.ToDTO(), nil
}

func (s *ReminderService) ListByUser(ctx context.Context, userID string) ([]*model.ReminderDTO, error) {
	reminders, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDTOs(reminders), nil
}

func (s *ReminderService) ListByJob(ctx context.Context, userID, jobID string) ([]*model.ReminderDTO, error) {
	reminders, err := s.repo.ListByJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	return toDTOs(reminders), nil
}

// Update applies a partial change to a reminder the user owns. It loads the
// existing reminder (ownership-checked) first, mutates only the provided fields,
// then persists — so an absent field is left untouched.
func (s *ReminderService) Update(ctx context.Context, userID, reminderID string, req *model.UpdateReminderRequest) (*model.ReminderDTO, error) {
	existing, err := s.repo.GetByID(ctx, userID, reminderID)
	if err != nil {
		return nil, err
	}

	if req.Message != nil {
		message := strings.TrimSpace(*req.Message)
		if message == "" {
			return nil, model.ErrMessageRequired
		}
		existing.Message = message
	}
	if req.RemindAt != nil {
		existing.RemindAt = *req.RemindAt
	}
	if req.IsDone != nil {
		existing.IsDone = *req.IsDone
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing.ToDTO(), nil
}

func (s *ReminderService) Delete(ctx context.Context, userID, reminderID string) error {
	return s.repo.Delete(ctx, userID, reminderID)
}

func toDTOs(reminders []*model.Reminder) []*model.ReminderDTO {
	dtos := make([]*model.ReminderDTO, len(reminders))
	for i, rem := range reminders {
		dtos[i] = rem.ToDTO()
	}
	return dtos
}
