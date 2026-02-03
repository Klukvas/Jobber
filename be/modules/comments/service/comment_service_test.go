package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommentRepository implements ports.CommentRepository
type MockCommentRepository struct {
	CreateFunc            func(ctx context.Context, comment *model.Comment) error
	ListByApplicationFunc func(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error)
	DeleteFunc            func(ctx context.Context, userID, commentID string) error
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) ListByApplication(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error) {
	if m.ListByApplicationFunc != nil {
		return m.ListByApplicationFunc(ctx, appID, userID...)
	}
	return nil, nil
}

func (m *MockCommentRepository) Delete(ctx context.Context, userID, commentID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, commentID)
	}
	return nil
}

func TestCommentService_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates comment successfully", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *model.Comment) error {
				comment.ID = "comment-1"
				comment.CreatedAt = time.Now()
				comment.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			ApplicationID: "app-1",
			Content:       "This is a comment",
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "comment-1", result.ID)
		assert.Equal(t, "This is a comment", result.Content)
		assert.Equal(t, "app-1", result.ApplicationID)
	})

	t.Run("returns error for empty content", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			ApplicationID: "app-1",
			Content:       "   ",
		}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrContentRequired, err)
	})

	t.Run("creates comment with stage ID", func(t *testing.T) {
		var createdComment *model.Comment
		stageID := "stage-1"

		mockRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *model.Comment) error {
				createdComment = comment
				comment.ID = "comment-1"
				return nil
			},
		}

		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			ApplicationID: "app-1",
			StageID:       &stageID,
			Content:       "Stage comment",
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, &stageID, createdComment.StageID)
	})

	t.Run("trims whitespace from content", func(t *testing.T) {
		var createdComment *model.Comment

		mockRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *model.Comment) error {
				createdComment = comment
				comment.ID = "comment-1"
				return nil
			},
		}

		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			ApplicationID: "app-1",
			Content:       "  Comment with whitespace  ",
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Comment with whitespace", createdComment.Content)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *model.Comment) error {
				return expectedError
			},
		}

		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			ApplicationID: "app-1",
			Content:       "Test comment",
		}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestCommentService_ListByApplication(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("returns comments list", func(t *testing.T) {
		stageID := "stage-1"
		expectedComments := []*model.Comment{
			{
				ID:            "comment-1",
				ApplicationID: appID,
				Content:       "First comment",
				CreatedAt:     time.Now(),
			},
			{
				ID:            "comment-2",
				ApplicationID: appID,
				StageID:       &stageID,
				Content:       "Second comment",
				CreatedAt:     time.Now(),
			},
		}

		mockRepo := &MockCommentRepository{
			ListByApplicationFunc: func(ctx context.Context, aid string, uid ...string) ([]*model.Comment, error) {
				assert.Equal(t, appID, aid)
				return expectedComments, nil
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByApplication(context.Background(), appID, userID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "First comment", result[0].Content)
		assert.Equal(t, "Second comment", result[1].Content)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			ListByApplicationFunc: func(ctx context.Context, aid string, uid ...string) ([]*model.Comment, error) {
				return []*model.Comment{}, nil
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByApplication(context.Background(), appID, userID)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockCommentRepository{
			ListByApplicationFunc: func(ctx context.Context, aid string, uid ...string) ([]*model.Comment, error) {
				return nil, expectedError
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByApplication(context.Background(), appID, userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestCommentService_Delete(t *testing.T) {
	userID := "user-123"
	commentID := "comment-1"

	t.Run("deletes comment successfully", func(t *testing.T) {
		var deletedCommentID string

		mockRepo := &MockCommentRepository{
			DeleteFunc: func(ctx context.Context, uid, cid string) error {
				deletedCommentID = cid
				return nil
			},
		}

		svc := NewCommentService(mockRepo)
		err := svc.Delete(context.Background(), userID, commentID)

		require.NoError(t, err)
		assert.Equal(t, commentID, deletedCommentID)
	})

	t.Run("returns error when comment not found", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			DeleteFunc: func(ctx context.Context, uid, cid string) error {
				return model.ErrCommentNotFound
			},
		}

		svc := NewCommentService(mockRepo)
		err := svc.Delete(context.Background(), userID, commentID)

		assert.Equal(t, model.ErrCommentNotFound, err)
	})
}

func TestComment_ToDTO(t *testing.T) {
	now := time.Now()
	stageID := "stage-1"

	comment := &model.Comment{
		ID:            "comment-1",
		UserID:        "user-123",
		ApplicationID: "app-1",
		StageID:       &stageID,
		Content:       "Test comment",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	dto := comment.ToDTO()

	assert.Equal(t, comment.ID, dto.ID)
	assert.Equal(t, comment.ApplicationID, dto.ApplicationID)
	assert.Equal(t, comment.StageID, dto.StageID)
	assert.Equal(t, comment.Content, dto.Content)
	assert.Equal(t, comment.CreatedAt, dto.CreatedAt)
}
