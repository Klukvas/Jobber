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
	CreateFunc    func(ctx context.Context, comment *model.Comment) error
	ListByJobFunc func(ctx context.Context, jobID string, userID ...string) ([]*model.Comment, error)
	UpdateFunc    func(ctx context.Context, userID, commentID, content string) (*model.Comment, error)
	DeleteFunc    func(ctx context.Context, userID, commentID string) error
}

func (m *MockCommentRepository) Update(ctx context.Context, userID, commentID, content string) (*model.Comment, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, userID, commentID, content)
	}
	return nil, nil
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) ListByJob(ctx context.Context, jobID, userID string) ([]*model.Comment, error) {
	if m.ListByJobFunc != nil {
		return m.ListByJobFunc(ctx, jobID, userID)
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
			JobID:   "job-1",
			Content: "This is a comment",
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "comment-1", result.ID)
		assert.Equal(t, "This is a comment", result.Content)
		assert.Equal(t, "job-1", result.JobID)
	})

	t.Run("returns error for empty content", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := NewCommentService(mockRepo)
		req := &model.CreateCommentRequest{
			JobID:   "job-1",
			Content: "   ",
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
			JobID:   "job-1",
			StageID: &stageID,
			Content: "Stage comment",
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
			JobID:   "job-1",
			Content: "  Comment with whitespace  ",
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
			JobID:   "job-1",
			Content: "Test comment",
		}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestCommentService_ListByJob(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("returns comments list", func(t *testing.T) {
		stageID := "stage-1"
		expectedComments := []*model.Comment{
			{
				ID:        "comment-1",
				JobID:     jobID,
				Content:   "First comment",
				CreatedAt: time.Now(),
			},
			{
				ID:        "comment-2",
				JobID:     jobID,
				StageID:   &stageID,
				Content:   "Second comment",
				CreatedAt: time.Now(),
			},
		}

		mockRepo := &MockCommentRepository{
			ListByJobFunc: func(ctx context.Context, jid string, uid ...string) ([]*model.Comment, error) {
				assert.Equal(t, jobID, jid)
				return expectedComments, nil
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByJob(context.Background(), jobID, userID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "First comment", result[0].Content)
		assert.Equal(t, "Second comment", result[1].Content)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			ListByJobFunc: func(ctx context.Context, jid string, uid ...string) ([]*model.Comment, error) {
				return []*model.Comment{}, nil
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByJob(context.Background(), jobID, userID)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockCommentRepository{
			ListByJobFunc: func(ctx context.Context, jid string, uid ...string) ([]*model.Comment, error) {
				return nil, expectedError
			},
		}

		svc := NewCommentService(mockRepo)
		result, err := svc.ListByJob(context.Background(), jobID, userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

// The id column is a uuid, so the tests have to use one: a placeholder string
// is now refused before it reaches storage, which is the point of validCommentID.
const validCommentID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"

func TestCommentService_Update(t *testing.T) {
	const (
		userID    = "user-123"
		commentID = validCommentID
	)

	tests := []struct {
		name        string
		content     string
		wantContent string
		wantErr     error
	}{
		{name: "rewrites the body", content: "Edited body", wantContent: "Edited body"},
		{name: "trims surrounding whitespace", content: "  Edited  ", wantContent: "Edited"},
		{name: "keeps internal line breaks", content: "line 1\nline 2", wantContent: "line 1\nline 2"},
		{name: "rejects empty content", content: "", wantErr: model.ErrContentRequired},
		{name: "rejects whitespace-only content", content: "   \n\t ", wantErr: model.ErrContentRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawUserID, sawCommentID, sawContent string
			mockRepo := &MockCommentRepository{
				UpdateFunc: func(_ context.Context, uid, cid, content string) (*model.Comment, error) {
					sawUserID, sawCommentID, sawContent = uid, cid, content
					return &model.Comment{ID: cid, UserID: uid, JobID: "job-1", Content: content}, nil
				},
			}

			svc := NewCommentService(mockRepo)
			dto, err := svc.Update(context.Background(), userID, commentID, &model.UpdateCommentRequest{Content: tt.content})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, dto)
				assert.Empty(t, sawContent, "the repository must not be called for invalid input")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, dto)
			assert.Equal(t, tt.wantContent, dto.Content)
			assert.Equal(t, tt.wantContent, sawContent)
			// The author is always part of the write, so the repository can
			// scope the UPDATE to rows this user owns.
			assert.Equal(t, userID, sawUserID)
			assert.Equal(t, commentID, sawCommentID)
		})
	}

	t.Run("propagates not-found from the ownership-scoped write", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
				return nil, model.ErrCommentNotFound
			},
		}

		svc := NewCommentService(mockRepo)
		dto, err := svc.Update(context.Background(), userID, commentID, &model.UpdateCommentRequest{Content: "hi there"})

		assert.ErrorIs(t, err, model.ErrCommentNotFound)
		assert.Nil(t, dto)
	})
}

// Same rule on the update path: a malformed id is a 404, and the content check
// still runs first so an empty body is reported as the empty body it is.
func TestCommentService_UpdateRejectsMalformedID(t *testing.T) {
	touched := false
	svc := NewCommentService(&MockCommentRepository{
		UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
			touched = true
			return nil, nil
		},
	})

	for _, id := range []string{"", "comment-1", "../../etc/passwd"} {
		dto, err := svc.Update(context.Background(), "user-123", id,
			&model.UpdateCommentRequest{Content: "Edited body"})

		assert.Nil(t, dto, "id %q", id)
		assert.ErrorIs(t, err, model.ErrCommentNotFound, "id %q", id)
	}
	assert.False(t, touched)

	_, err := svc.Update(context.Background(), "user-123", "comment-1",
		&model.UpdateCommentRequest{Content: "   "})
	assert.ErrorIs(t, err, model.ErrContentRequired)
}

func TestCommentService_Delete(t *testing.T) {
	userID := "user-123"
	commentID := validCommentID

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

	// A path parameter that is not a uuid used to reach Postgres as a cast
	// error and come back as a 500 — our failure, for what is plainly a bad
	// request. There is no such comment, which is the same 404 as any other id
	// that matches nothing.
	t.Run("answers a malformed id as not found, without touching storage", func(t *testing.T) {
		for _, id := range []string{"", "comment-1", "../../etc/passwd", "7c9e6679-7425-40de-944b"} {
			touched := false
			svc := NewCommentService(&MockCommentRepository{
				DeleteFunc: func(context.Context, string, string) error {
					touched = true
					return nil
				},
			})

			err := svc.Delete(context.Background(), userID, id)

			assert.ErrorIs(t, err, model.ErrCommentNotFound, "id %q", id)
			assert.False(t, touched, "id %q", id)
		}
	})
}

func TestComment_ToDTO(t *testing.T) {
	now := time.Now()
	stageID := "stage-1"

	comment := &model.Comment{
		ID:        "comment-1",
		UserID:    "user-123",
		JobID:     "job-1",
		StageID:   &stageID,
		Content:   "Test comment",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := comment.ToDTO()

	assert.Equal(t, comment.ID, dto.ID)
	assert.Equal(t, comment.JobID, dto.JobID)
	assert.Equal(t, comment.StageID, dto.StageID)
	assert.Equal(t, comment.Content, dto.Content)
	assert.Equal(t, comment.CreatedAt, dto.CreatedAt)
}
