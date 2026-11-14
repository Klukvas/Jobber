package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so a
// comment id must reach the repository in canonical form or the write is a 500.
func TestCommentService_ForwardsCanonicalCommentID(t *testing.T) {
	const commentID = "6f1c2d3e-4b5a-4c6d-8e7f-90a1b2c3d4e5"

	for name, form := range idstest.NonCanonicalForms(commentID) {
		t.Run(name, func(t *testing.T) {
			var updated, deleted string
			svc := NewCommentService(&MockCommentRepository{
				UpdateFunc: func(_ context.Context, _, id, _ string) (*model.Comment, error) {
					updated = id
					return &model.Comment{ID: id}, nil
				},
				DeleteFunc: func(_ context.Context, _, id string) error {
					deleted = id
					return nil
				},
			})

			_, err := svc.Update(context.Background(), "user-123", form, &model.UpdateCommentRequest{Content: "Edited"})
			require.NoError(t, err)
			require.NoError(t, svc.Delete(context.Background(), "user-123", form))

			assert.Equal(t, commentID, updated)
			assert.Equal(t, commentID, deleted)
		})
	}
}
