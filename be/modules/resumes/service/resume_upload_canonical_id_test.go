package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The resume id becomes part of the storage key and the row id. uuid.Parse also
// accepts spellings such as `urn:uuid:…`, which Postgres rejects and which
// would name a different object, so only the canonical id may be used.
func TestResumeService_FinalizeUpload_UsesCanonicalID(t *testing.T) {
	const (
		userID    = "user-123"
		resumeID  = "3f2a6c1e-0000-4000-8000-00000000abcd"
		objectKey = "users/user-123/resumes/3f2a6c1e-0000-4000-8000-00000000abcd.pdf"
	)

	for name, form := range idstest.NonCanonicalForms(resumeID) {
		t.Run(name, func(t *testing.T) {
			objects := map[string][]byte{objectKey: pdfBytes(2048)}
			s3Client, cleanup := storage.NewTestS3Client(objects)
			defer cleanup()

			var created *model.Resume
			var looked []string
			repo := &MockResumeRepository{
				CreateFunc: func(_ context.Context, r *model.Resume) error {
					created = r
					return nil
				},
				GetByIDFunc: func(_ context.Context, _, id string) (*model.Resume, error) {
					looked = append(looked, id)
					return nil, model.ErrResumeNotFound
				},
			}
			svc := NewResumeService(repo, s3Client, nil, nil)

			_, err := svc.FinalizeUpload(context.Background(), userID, form, &model.FinalizeUploadRequest{})

			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, resumeID, created.ID)
			assert.Equal(t, objectKey, *created.StorageKey)
			for _, id := range looked {
				assert.Equal(t, resumeID, id)
			}
		})
	}
}
