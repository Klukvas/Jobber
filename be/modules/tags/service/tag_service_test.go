package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTagRepository implements ports.TagRepository.
type MockTagRepository struct {
	CreateFunc         func(ctx context.Context, tag *model.Tag) error
	ListFunc           func(ctx context.Context, userID string) ([]*model.Tag, error)
	DeleteFunc         func(ctx context.Context, userID, tagID string) error
	AddRelationFunc    func(ctx context.Context, userID string, rel *model.TagRelation) error
	RemoveRelationFunc func(ctx context.Context, userID, tagID, entityType, entityID string) error
	ListByEntityFunc   func(ctx context.Context, userID, entityType, entityID string) ([]*model.Tag, error)
}

func (m *MockTagRepository) Create(ctx context.Context, tag *model.Tag) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, tag)
	}
	return nil
}
func (m *MockTagRepository) List(ctx context.Context, userID string) ([]*model.Tag, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}
func (m *MockTagRepository) Delete(ctx context.Context, userID, tagID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, tagID)
	}
	return nil
}
func (m *MockTagRepository) AddRelation(ctx context.Context, userID string, rel *model.TagRelation) error {
	if m.AddRelationFunc != nil {
		return m.AddRelationFunc(ctx, userID, rel)
	}
	return nil
}
func (m *MockTagRepository) RemoveRelation(ctx context.Context, userID, tagID, entityType, entityID string) error {
	if m.RemoveRelationFunc != nil {
		return m.RemoveRelationFunc(ctx, userID, tagID, entityType, entityID)
	}
	return nil
}
func (m *MockTagRepository) ListByEntity(ctx context.Context, userID, entityType, entityID string) ([]*model.Tag, error) {
	if m.ListByEntityFunc != nil {
		return m.ListByEntityFunc(ctx, userID, entityType, entityID)
	}
	return nil, nil
}

func TestTagService_Create(t *testing.T) {
	userID := "user-1"

	t.Run("creates a tag and trims the name", func(t *testing.T) {
		var captured *model.Tag
		repo := &MockTagRepository{
			CreateFunc: func(_ context.Context, tag *model.Tag) error {
				tag.ID = "tag-1"
				captured = tag
				return nil
			},
		}
		svc := NewTagService(repo)

		dto, err := svc.Create(context.Background(), userID, &model.CreateTagRequest{Name: "  urgent  "})

		require.NoError(t, err)
		assert.Equal(t, "urgent", dto.Name)
		assert.Equal(t, userID, captured.UserID)
	})

	t.Run("rejects a blank name", func(t *testing.T) {
		svc := NewTagService(&MockTagRepository{})
		_, err := svc.Create(context.Background(), userID, &model.CreateTagRequest{Name: "   "})
		assert.ErrorIs(t, err, model.ErrTagNameRequired)
	})

	t.Run("accepts a valid hex color", func(t *testing.T) {
		repo := &MockTagRepository{CreateFunc: func(_ context.Context, tag *model.Tag) error { tag.ID = "t"; return nil }}
		svc := NewTagService(repo)
		color := "#2563EB"
		_, err := svc.Create(context.Background(), userID, &model.CreateTagRequest{Name: "x", Color: &color})
		require.NoError(t, err)
	})

	t.Run("rejects a malformed color", func(t *testing.T) {
		svc := NewTagService(&MockTagRepository{})
		bad := "blue"
		_, err := svc.Create(context.Background(), userID, &model.CreateTagRequest{Name: "x", Color: &bad})
		assert.ErrorIs(t, err, model.ErrInvalidColor)
	})

	t.Run("propagates duplicate-name conflict from repo", func(t *testing.T) {
		repo := &MockTagRepository{CreateFunc: func(_ context.Context, _ *model.Tag) error { return model.ErrTagNameExists }}
		svc := NewTagService(repo)
		_, err := svc.Create(context.Background(), userID, &model.CreateTagRequest{Name: "dup"})
		assert.ErrorIs(t, err, model.ErrTagNameExists)
	})
}

func TestTagService_Relations(t *testing.T) {
	userID := "user-1"

	t.Run("attach rejects an invalid entity type before hitting the repo", func(t *testing.T) {
		called := false
		repo := &MockTagRepository{AddRelationFunc: func(_ context.Context, _ string, _ *model.TagRelation) error {
			called = true
			return nil
		}}
		svc := NewTagService(repo)

		err := svc.Attach(context.Background(), userID, "tag-1", &model.AttachTagRequest{EntityType: "resume", EntityID: "e1"})

		assert.ErrorIs(t, err, model.ErrInvalidEntityType)
		assert.False(t, called)
	})

	t.Run("attach forwards a valid job relation with the userID", func(t *testing.T) {
		var gotUser string
		var gotRel *model.TagRelation
		repo := &MockTagRepository{AddRelationFunc: func(_ context.Context, u string, rel *model.TagRelation) error {
			gotUser, gotRel = u, rel
			return nil
		}}
		svc := NewTagService(repo)

		err := svc.Attach(context.Background(), userID, "tag-1", &model.AttachTagRequest{EntityType: "job", EntityID: "job-9"})

		require.NoError(t, err)
		assert.Equal(t, userID, gotUser)
		assert.Equal(t, "tag-1", gotRel.TagID)
		assert.Equal(t, "job", gotRel.EntityType)
		assert.Equal(t, "job-9", gotRel.EntityID)
	})

	t.Run("attach propagates entity-not-found from the repo (ownership check)", func(t *testing.T) {
		repo := &MockTagRepository{AddRelationFunc: func(_ context.Context, _ string, _ *model.TagRelation) error {
			return model.ErrEntityNotFound
		}}
		svc := NewTagService(repo)
		err := svc.Attach(context.Background(), userID, "tag-1", &model.AttachTagRequest{EntityType: "company", EntityID: "not-mine"})
		assert.ErrorIs(t, err, model.ErrEntityNotFound)
	})

	t.Run("listByEntity rejects an invalid entity type", func(t *testing.T) {
		svc := NewTagService(&MockTagRepository{})
		_, err := svc.ListByEntity(context.Background(), userID, "resume", "e1")
		assert.ErrorIs(t, err, model.ErrInvalidEntityType)
	})

	t.Run("listByEntity maps tags to DTOs", func(t *testing.T) {
		repo := &MockTagRepository{ListByEntityFunc: func(_ context.Context, _, _, _ string) ([]*model.Tag, error) {
			return []*model.Tag{{ID: "a"}, {ID: "b"}}, nil
		}}
		svc := NewTagService(repo)
		dtos, err := svc.ListByEntity(context.Background(), userID, "job", "job-1")
		require.NoError(t, err)
		assert.Len(t, dtos, 2)
	})
}
