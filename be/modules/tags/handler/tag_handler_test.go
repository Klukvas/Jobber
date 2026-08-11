package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/andreypavlenko/jobber/modules/tags/service"
	"github.com/gin-gonic/gin"
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
	return m.CreateFunc(ctx, tag)
}
func (m *MockTagRepository) List(ctx context.Context, userID string) ([]*model.Tag, error) {
	return m.ListFunc(ctx, userID)
}
func (m *MockTagRepository) Delete(ctx context.Context, userID, tagID string) error {
	return m.DeleteFunc(ctx, userID, tagID)
}
func (m *MockTagRepository) AddRelation(ctx context.Context, userID string, rel *model.TagRelation) error {
	return m.AddRelationFunc(ctx, userID, rel)
}
func (m *MockTagRepository) RemoveRelation(ctx context.Context, userID, tagID, entityType, entityID string) error {
	return m.RemoveRelationFunc(ctx, userID, tagID, entityType, entityID)
}
func (m *MockTagRepository) ListByEntity(ctx context.Context, userID, entityType, entityID string) ([]*model.Tag, error) {
	return m.ListByEntityFunc(ctx, userID, entityType, entityID)
}

func newTestRouter(repo *MockTagRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewTagService(repo)
	h := NewTagHandler(svc)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	h.RegisterRoutes(v1, func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	return engine
}

func do(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestTagHandler_CreateAndList(t *testing.T) {
	t.Run("201 on create", func(t *testing.T) {
		repo := &MockTagRepository{CreateFunc: func(_ context.Context, tag *model.Tag) error {
			tag.ID = "tag-1"
			return nil
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodPost, "/api/v1/tags", `{"name":"urgent","color":"#2563eb"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		var dto model.TagDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "urgent", dto.Name)
	})

	t.Run("409 on duplicate name", func(t *testing.T) {
		repo := &MockTagRepository{CreateFunc: func(_ context.Context, _ *model.Tag) error {
			return model.ErrTagNameExists
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodPost, "/api/v1/tags", `{"name":"dup"}`)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("400 on missing name", func(t *testing.T) {
		engine := newTestRouter(&MockTagRepository{})
		w := do(engine, http.MethodPost, "/api/v1/tags", `{"color":"#2563eb"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("200 on list", func(t *testing.T) {
		repo := &MockTagRepository{ListFunc: func(_ context.Context, _ string) ([]*model.Tag, error) {
			return []*model.Tag{{ID: "a"}, {ID: "b"}}, nil
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodGet, "/api/v1/tags", "")

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []model.TagDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 2)
	})
}

func TestTagHandler_Relations(t *testing.T) {
	t.Run("200 attaching a tag to a job", func(t *testing.T) {
		var gotRel *model.TagRelation
		repo := &MockTagRepository{AddRelationFunc: func(_ context.Context, _ string, rel *model.TagRelation) error {
			gotRel = rel
			return nil
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodPost, "/api/v1/tags/tag-1/relations", `{"entity_type":"job","entity_id":"job-9"}`)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "job-9", gotRel.EntityID)
	})

	t.Run("400 attaching with an invalid entity_type (binding)", func(t *testing.T) {
		engine := newTestRouter(&MockTagRepository{})
		w := do(engine, http.MethodPost, "/api/v1/tags/tag-1/relations", `{"entity_type":"resume","entity_id":"x"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("404 when the entity is not owned", func(t *testing.T) {
		repo := &MockTagRepository{AddRelationFunc: func(_ context.Context, _ string, _ *model.TagRelation) error {
			return model.ErrEntityNotFound
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodPost, "/api/v1/tags/tag-1/relations", `{"entity_type":"job","entity_id":"nope"}`)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("200 detaching with query params", func(t *testing.T) {
		var gotType, gotID string
		repo := &MockTagRepository{RemoveRelationFunc: func(_ context.Context, _, _, entityType, entityID string) error {
			gotType, gotID = entityType, entityID
			return nil
		}}
		engine := newTestRouter(repo)

		w := do(engine, http.MethodDelete, "/api/v1/tags/tag-1/relations?entity_type=job&entity_id=job-9", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "job", gotType)
		assert.Equal(t, "job-9", gotID)
	})

	t.Run("400 detaching without entity_id", func(t *testing.T) {
		engine := newTestRouter(&MockTagRepository{})
		w := do(engine, http.MethodDelete, "/api/v1/tags/tag-1/relations?entity_type=job", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("200 listing tags for a job and for a company", func(t *testing.T) {
		repo := &MockTagRepository{ListByEntityFunc: func(_ context.Context, _, entityType, _ string) ([]*model.Tag, error) {
			return []*model.Tag{{ID: entityType}}, nil
		}}
		engine := newTestRouter(repo)

		wJob := do(engine, http.MethodGet, "/api/v1/jobs/job-1/tags", "")
		wCompany := do(engine, http.MethodGet, "/api/v1/companies/co-1/tags", "")

		assert.Equal(t, http.StatusOK, wJob.Code)
		assert.Equal(t, http.StatusOK, wCompany.Code)
	})
}
