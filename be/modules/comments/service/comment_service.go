package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/comments/ports"
)

type CommentService struct {
	repo ports.CommentRepository
}

func NewCommentService(repo ports.CommentRepository) *CommentService {
	return &CommentService{repo: repo}
}

func (s *CommentService) Create(ctx context.Context, userID string, req *model.CreateCommentRequest) (*model.CommentDTO, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, model.ErrContentRequired
	}

	comment := &model.Comment{
		UserID:        userID,
		ApplicationID: req.ApplicationID,
		StageID:       req.StageID,
		Content:       strings.TrimSpace(req.Content),
	}

	if err := s.repo.Create(ctx, comment); err != nil {
		return nil, err
	}
	return comment.ToDTO(), nil
}

func (s *CommentService) ListByApplication(ctx context.Context, appID string, userID ...string) ([]*model.CommentDTO, error) {
	comments, err := s.repo.ListByApplication(ctx, appID, userID...)
	if err != nil {
		return nil, err
	}

	dtos := make([]*model.CommentDTO, len(comments))
	for i, comment := range comments {
		dtos[i] = comment.ToDTO()
	}
	return dtos, nil
}

func (s *CommentService) Delete(ctx context.Context, userID, commentID string) error {
	return s.repo.Delete(ctx, userID, commentID)
}
