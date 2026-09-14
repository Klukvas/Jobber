package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pdfBytes(size int) []byte {
	out := make([]byte, size)
	copy(out, "%PDF-1.7\n")
	return out
}

func TestValidateUploadedPDF(t *testing.T) {
	tests := []struct {
		name    string
		prefix  []byte
		size    int64
		wantErr error
	}{
		{
			name:   "a real pdf passes",
			prefix: []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3"),
			size:   4096,
		},
		{
			name:    "a text file renamed .pdf is rejected",
			prefix:  []byte("this is not a pdf, it is a text file\n"),
			size:    36,
			wantErr: model.ErrInvalidFileContent,
		},
		{
			name:    "a zip disguised as a pdf is rejected",
			prefix:  []byte("PK\x03\x04"),
			size:    2048,
			wantErr: model.ErrInvalidFileContent,
		},
		{
			name:    "a pdf marker that is not at the start is rejected",
			prefix:  []byte("GIF89a%PDF-1.4"),
			size:    14,
			wantErr: model.ErrInvalidFileContent,
		},
		{
			name:    "an empty object means the upload never landed",
			prefix:  nil,
			size:    0,
			wantErr: model.ErrResumeFileMissing,
		},
		{
			name:    "a negative size is treated as missing",
			prefix:  []byte("%PDF-"),
			size:    -1,
			wantErr: model.ErrResumeFileMissing,
		},
		{
			name:   "exactly at the size cap passes",
			prefix: []byte("%PDF-1.7"),
			size:   MaxUploadedResumeBytes,
		},
		{
			name:    "one byte over the size cap is rejected",
			prefix:  []byte("%PDF-1.7"),
			size:    MaxUploadedResumeBytes + 1,
			wantErr: model.ErrFileTooLarge,
		},
		{
			name:    "a truncated magic number is rejected",
			prefix:  []byte("%PD"),
			size:    3,
			wantErr: model.ErrInvalidFileContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadedPDF(tt.prefix, tt.size)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// storedUpload is the row a successful finalize leaves behind.
func storedUpload(userID, resumeID, storageKey, title string) *model.Resume {
	key := storageKey
	return &model.Resume{
		ID:          resumeID,
		UserID:      userID,
		Title:       title,
		StorageType: model.StorageTypeS3,
		StorageKey:  &key,
		IsActive:    true,
	}
}

// placeholderUpload is the row the presign-era upload flow wrote: an inactive
// S3 resume with no file_url and no edit since it was created.
func placeholderUpload(userID, resumeID, storageKey string) *model.Resume {
	key := storageKey
	created := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return &model.Resume{
		ID:          resumeID,
		UserID:      userID,
		Title:       "Untitled Resume",
		StorageType: model.StorageTypeS3,
		StorageKey:  &key,
		IsActive:    false,
		CreatedAt:   created,
		UpdatedAt:   created,
	}
}

func TestResumeService_FinalizeUpload(t *testing.T) {
	const (
		userID    = "user-123"
		resumeID  = "3f2a6c1e-0000-4000-8000-00000000abcd"
		objectKey = "users/user-123/resumes/3f2a6c1e-0000-4000-8000-00000000abcd.pdf"
	)

	t.Run("creates the resume with the given title when the object is a PDF", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		var created *model.Resume
		repo := &MockResumeRepository{
			CreateFunc: func(_ context.Context, r *model.Resume) error {
				created = r
				return nil
			},
			// The idempotency lookup runs first and finds nothing, so this is a
			// genuinely new upload.
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		title := "Backend Engineer"
		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, &model.FinalizeUploadRequest{Title: &title})

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "Backend Engineer", dto.Title)
		assert.True(t, dto.IsActive, "a verified upload is immediately usable")
		assert.Equal(t, objectKey, *created.StorageKey)
		assert.Equal(t, userID, created.UserID)
		assert.Contains(t, objects, objectKey, "a valid upload must stay in storage")
	})

	t.Run("falls back to the placeholder title when none is supplied", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		svc := NewResumeService(&MockResumeRepository{}, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, &model.FinalizeUploadRequest{})

		require.NoError(t, err)
		assert.Equal(t, "Untitled Resume", dto.Title)
		assert.True(t, dto.IsActive)
	})

	// The whole point of deferring row creation: a rejected or abandoned upload
	// must leave nothing that can be listed, counted against the plan, or served.
	t.Run("rejects a spoofed .pdf, writes no row and removes the object", func(t *testing.T) {
		objects := map[string][]byte{objectKey: []byte("just some text pretending to be a resume")}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrInvalidFileContent)
		assert.Nil(t, dto)
		assert.False(t, created, "a rejected upload must never become a resume")
		assert.NotContains(t, objects, objectKey, "the rejected object must be removed")
	})

	// A PUT that stored zero bytes leaves a real object behind, and storage
	// answers a ranged read of it with 416, not 404. That has to reach the size
	// check and come back as a plain "no file", never as a 500.
	t.Run("rejects a present but empty object as a missing file", func(t *testing.T) {
		objects := map[string][]byte{objectKey: {}}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeFileMissing)
		assert.Equal(t, model.CodeResumeFileMissing, model.GetErrorCode(err),
			"an empty upload is the customer's 400, not our 500")
		assert.Nil(t, dto)
		assert.False(t, created, "an empty upload must not become a resume")
		assert.NotContains(t, objects, objectKey,
			"nothing references the empty object, so it is cleaned up")
	})

	// The size cap is only as good as the size it is given. A store that
	// answers the ranged read with an unknown total ("bytes 0-1023/*") used to
	// make an oversized object look like the 1 KB that was actually read, and
	// the cap waved it through.
	t.Run("rejects an oversized object even when the range response hides its size", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(MaxUploadedResumeBytes + 1)}
		s3Client, cleanup := storage.NewTestS3ClientWithOptions(objects, storage.TestS3Options{
			ContentRangeOverride: map[string]string{objectKey: "bytes 0-1023/*"},
		})
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrFileTooLarge)
		assert.Nil(t, dto)
		assert.False(t, created, "an oversized upload must never become a resume")
	})

	// The size cannot be established at all: that is our failure, not a verdict
	// on the file, so it must not come back as a successful small upload.
	t.Run("fails closed when the object's size cannot be determined", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(MaxUploadedResumeBytes + 1)}
		s3Client, cleanup := storage.NewTestS3ClientWithOptions(objects, storage.TestS3Options{
			ContentRangeOverride: map[string]string{objectKey: "bytes 0-1023/*"},
			FailHead:             map[string]bool{objectKey: true},
		})
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.Error(t, err)
		assert.Nil(t, dto)
		assert.False(t, created)
		assert.NotErrorIs(t, err, model.ErrResumeFileMissing,
			"a store that will not report a size is our problem, not a missing upload")
		assert.Contains(t, objects, objectKey,
			"an unverified object must not be deleted on our own failure")
	})

	t.Run("rejects an upload that never landed, without creating a row", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeFileMissing)
		assert.False(t, created)
	})

	// The storage key is rebuilt from the authenticated user, so another user's
	// resume id names a path that does not exist for the caller.
	t.Run("cannot reach another user's object", func(t *testing.T) {
		const victimKey = "users/victim/resumes/3f2a6c1e-0000-4000-8000-00000000abcd.pdf"
		objects := map[string][]byte{victimKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), "attacker", resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeFileMissing)
		assert.False(t, created)
		assert.Contains(t, objects, victimKey, "another user's object must be untouched")
	})

	t.Run("refuses a resume id that is not the opaque identifier we issued", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		svc := NewResumeService(&MockResumeRepository{}, s3Client, nil, nil)

		for _, id := range []string{"../../etc/passwd", "not-a-uuid", ""} {
			_, err := svc.FinalizeUpload(context.Background(), userID, id, nil)
			assert.ErrorIs(t, err, model.ErrResumeFileMissing, "id %q", id)
		}
	})

	t.Run("enforces the plan limit at finalization, not only at presign", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		limit := &mockLimitChecker{err: errors.New("limit reached")}
		svc := NewResumeService(repo, s3Client, limit, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Error(t, err)
		assert.False(t, created)
	})

	// An INSERT can commit and still report a failure — a context deadline that
	// lands between the commit and the response, a connection dropped on the way
	// back. Deleting the object on that signal would strip the file out from
	// under a row that is live, so an ambiguous write must leave storage alone.
	t.Run("keeps the object when the row write fails ambiguously", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				return context.DeadlineExceeded
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeFileMissing,
			"a write failure is ours, not a missing upload")
		assert.Contains(t, objects, objectKey,
			"the object may already be referenced by a committed row")
	})

	t.Run("keeps the object on any other row write failure", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				return errors.New("boom")
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Error(t, err)
		assert.Contains(t, objects, objectKey)
	})

	// Retrying a finalize that already succeeded — a dropped response, a
	// double-tap — has to return the resume, not fail and not delete its file.
	t.Run("is idempotent: a repeated finalize returns the stored resume", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		stored := storedUpload(userID, resumeID, objectKey, "Backend Engineer")
		createCalls := 0
		repo := &MockResumeRepository{
			GetByIDFunc: func(_ context.Context, uid, id string) (*model.Resume, error) {
				assert.Equal(t, userID, uid, "the lookup must be scoped to the caller")
				assert.Equal(t, resumeID, id)
				return stored, nil
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				createCalls++
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		title := "A Different Title"
		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, &model.FinalizeUploadRequest{Title: &title})

		require.NoError(t, err)
		assert.Equal(t, "Backend Engineer", dto.Title, "the first finalize owns the title")
		assert.Equal(t, 0, createCalls, "the row already exists")
		assert.Contains(t, objects, objectKey, "the live object must survive a retry")
	})

	t.Run("a retry succeeds even when the stored row fills the last plan slot", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return storedUpload(userID, resumeID, objectKey, "Backend Engineer"), nil
			},
		}
		limit := &mockLimitChecker{err: errors.New("limit reached")}
		svc := NewResumeService(repo, s3Client, limit, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.NoError(t, err)
		assert.Equal(t, resumeID, dto.ID)
	})

	// Two finalizations in flight at once: one INSERT wins, the loser gets a
	// unique violation and must hand back the winner's row.
	t.Run("resolves a concurrent finalize through the unique violation", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		rowExists := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				if !rowExists {
					return nil, model.ErrResumeNotFound
				}
				return storedUpload(userID, resumeID, objectKey, "Backend Engineer"), nil
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				// The winner committed between our lookup and our INSERT.
				rowExists = true
				return model.ErrResumeAlreadyExists
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.NoError(t, err)
		assert.Equal(t, "Backend Engineer", dto.Title)
		assert.Contains(t, objects, objectKey, "the winner's object must not be deleted")
	})

	t.Run("refuses to hand over a row that is not this upload", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		// Same id, but an external-URL resume — not the object we just verified.
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return &model.Resume{
					ID: resumeID, UserID: userID, Title: "Linked CV",
					StorageType: model.StorageTypeExternal,
				}, nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		assert.Contains(t, objects, objectKey)
	})

	t.Run("refuses a stored row whose key is not the one we derived", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return storedUpload(userID, resumeID, "users/someone-else/resumes/x.pdf", "Sneaky"), nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
	})

	// A storage read that fails for any reason other than "no such key" is an
	// internal fault; reporting it as a missing upload blamed the customer.
	t.Run("does not report a transient storage failure as a missing file", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
		cleanup() // server is down: reads fail with a transport error, not a 404

		created := false
		repo := &MockResumeRepository{
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeFileMissing)
		assert.Equal(t, model.CodeInternalError, model.GetErrorCode(err),
			"an unreachable store must map to a 500, not a 400")
		assert.False(t, created)
	})

	t.Run("fails clearly when storage is not configured", func(t *testing.T) {
		svc := NewResumeService(&MockResumeRepository{}, nil, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Error(t, err)
	})

	t.Run("rejects a blank title rather than storing one", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		svc := NewResumeService(&MockResumeRepository{}, s3Client, nil, nil)

		blank := "   "
		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, &model.FinalizeUploadRequest{Title: &blank})

		assert.ErrorIs(t, err, model.ErrResumeTitleRequired)
		// The file itself is fine — this is a client sending a blank title, and
		// deleting somebody's upload over that is not this function's call. The
		// object is unreferenced, so the reclamation pass removes it later.
		assert.Contains(t, objects, objectKey)
	})

	/*
		The presign-era upload flow wrote its row before the object existed, so
		some customers still have one sitting under the id they are finalizing —
		with a real, valid PDF behind it, uploaded at the time.

		Every finalize of those answered 404 and always would: the row is a
		placeholder, so the idempotent lookup rightly refused to call it a
		finished upload, the INSERT could only collide with it, and the
		re-lookup saw the same placeholder again. The row is this upload's own,
		so it is handed to the write to claim rather than treated as somebody
		else's.
	*/
	t.Run("finalizes an upload the presign era left a placeholder for", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		var written *model.Resume
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return placeholderUpload(userID, resumeID, objectKey), nil
			},
			CreateFinalizedUploadFunc: func(_ context.Context, r *model.Resume, _ int) error {
				written = r
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		title := "Backend Engineer"
		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, &model.FinalizeUploadRequest{Title: &title})

		require.NoError(t, err)
		require.NotNil(t, written, "the placeholder must be handed to the write")
		assert.Equal(t, resumeID, written.ID)
		assert.Equal(t, userID, written.UserID)
		assert.Equal(t, objectKey, *written.StorageKey)
		assert.True(t, written.IsActive, "the claimed row is a usable resume")
		assert.Equal(t, "Backend Engineer", dto.Title)
		assert.Contains(t, objects, objectKey, "the customer's file must survive")
	})

	// The placeholder is a row that names this object. Deleting the file out
	// from under it would make an upload that is merely blocked by a full plan
	// unrecoverable — an upgrade would find nothing to finalize.
	t.Run("keeps a placeholder's object when the plan is full", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return placeholderUpload(userID, resumeID, objectKey), nil
			},
			CreateFinalizedUploadFunc: func(context.Context, *model.Resume, int) error {
				return model.ErrResumeLimitReached
			},
		}
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: 3}, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeLimitReached)
		assert.Contains(t, objects, objectKey)
	})

	// With no row naming it, an object refused for the plan limit is
	// unreferenced and there is nothing left that could ever finalize it.
	t.Run("removes an unreferenced object when the plan is full", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFinalizedUploadFunc: func(context.Context, *model.Resume, int) error {
				return model.ErrResumeLimitReached
			},
		}
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: 3}, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeLimitReached)
		assert.NotContains(t, objects, objectKey)
	})

	// A write that failed for a reason nobody can name may still have
	// committed. Deleting the object would strip the file out from under a live
	// row, and the customer has no way to get it back.
	t.Run("never deletes a valid object when the write fails ambiguously", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFinalizedUploadFunc: func(context.Context, *model.Resume, int) error {
				return errors.New("connection reset crossing the commit")
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.Error(t, err)
		assert.Equal(t, model.CodeInternalError, model.GetErrorCode(err))
		assert.Contains(t, objects, objectKey)
	})

	// The id is taken by a row this caller cannot see — somebody else's. The
	// answer is 404, and nothing of theirs is touched.
	t.Run("answers 404 for an id that belongs to another account", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := &MockResumeRepository{
			// Scoped to the caller, so another account's row simply is not there.
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFinalizedUploadFunc: func(context.Context, *model.Resume, int) error {
				return model.ErrResumeAlreadyExists
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		assert.Contains(t, objects, objectKey, "another account's object is not ours to delete")
	})

	// The rule for deleting is narrow on purpose: the object goes only when it
	// has been proved unusable, or when no row for it can ever exist. A failure
	// on our side is neither, and an upload must survive one.
	t.Run("keeps the object when the plan allowance cannot be read", func(t *testing.T) {
		objects := map[string][]byte{objectKey: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		created := false
		repo := &MockResumeRepository{
			CreateFinalizedUploadFunc: func(context.Context, *model.Resume, int) error {
				created = true
				return nil
			},
		}
		limit := &mockLimitChecker{err: errors.New("subscriptions are unreachable")}
		svc := NewResumeService(repo, s3Client, limit, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Error(t, err)
		assert.False(t, created)
		assert.Contains(t, objects, objectKey, "our own outage must not destroy the upload")
	})
}

// GenerateUploadURL must not reserve anything: the row it used to insert was
// the source of ghost resumes that consumed a plan slot forever.
func TestResumeService_GenerateUploadURL_CreatesNoRow(t *testing.T) {
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
	defer cleanup()

	created := false
	repo := &MockResumeRepository{
		CreateFunc: func(context.Context, *model.Resume) error {
			created = true
			return nil
		},
	}
	svc := NewResumeService(repo, s3Client, nil, nil)

	resp, err := svc.GenerateUploadURL(context.Background(), "user-123", &model.GenerateUploadURLRequest{
		Filename:    "cv.pdf",
		ContentType: "application/pdf",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.ResumeID)
	assert.NotEmpty(t, resp.UploadURL)
	assert.False(t, created, "presigning must not create a resume")
}

type mockLimitChecker struct {
	err error
	// max is the plan ceiling handed to the repository; -1 (the zero value is
	// deliberately not used) means unlimited.
	max int
}

func (m *mockLimitChecker) CheckLimit(context.Context, string, string) error { return m.err }

func (m *mockLimitChecker) ResourceLimit(context.Context, string, string) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.max, nil
}
