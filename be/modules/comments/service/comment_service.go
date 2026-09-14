package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/comments/ports"
	"github.com/google/uuid"
)

// isNoSuchComment reports whether an id cannot name a comment at all.
//
// The path parameter is caller-supplied text and `comments.id` is a uuid
// column, so anything that is not a uuid reaches Postgres as a cast error and
// comes back as a 500 — an internal failure blamed on us for what is plainly a
// bad request. There is no such comment and there never could be, which is a
// 404 like any other id that matches nothing.
func isNoSuchComment(commentID string) bool {
	_, err := uuid.Parse(commentID)
	return err != nil
}

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
		UserID:  userID,
		JobID:   req.JobID,
		StageID: req.StageID,
		Content: strings.TrimSpace(req.Content),
	}

	if err := s.repo.Create(ctx, comment); err != nil {
		return nil, err
	}
	return comment.ToDTO(), nil
}

func (s *CommentService) ListByJob(ctx context.Context, jobID, userID string) ([]*model.CommentDTO, error) {
	comments, err := s.repo.ListByJob(ctx, jobID, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*model.CommentDTO, len(comments))
	for i, comment := range comments {
		dtos[i] = comment.ToDTO()
	}
	return dtos, nil
}

// Update rewrites a comment the caller authored. Ownership is enforced by the
// repository write itself; this layer only validates the new content.
func (s *CommentService) Update(ctx context.Context, userID, commentID string, req *model.UpdateCommentRequest) (*model.CommentDTO, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, model.ErrContentRequired
	}
	if isNoSuchComment(commentID) {
		return nil, model.ErrCommentNotFound
	}

	comment, err := s.repo.Update(ctx, userID, commentID, content)
	if err != nil {
		return nil, err
	}
	return comment.ToDTO(), nil
}

func (s *CommentService) Delete(ctx context.Context, userID, commentID string) error {
	if isNoSuchComment(commentID) {
		return model.ErrCommentNotFound
	}
	return s.repo.Delete(ctx, userID, commentID)
}
