package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slotEnforcingRepo stands in for the transaction the real repository runs: it
// counts and inserts while holding a lock, which is exactly the property that
// makes the limit hold under concurrency. Everything else is a no-op.
type slotEnforcingRepo struct {
	mu    sync.Mutex
	rows  map[string]*model.Resume
	calls int
}

func newSlotEnforcingRepo() *slotEnforcingRepo {
	return &slotEnforcingRepo{rows: map[string]*model.Resume{}}
}

func (r *slotEnforcingRepo) Create(context.Context, *model.Resume) error { return nil }

func (r *slotEnforcingRepo) CreateFinalizedUpload(_ context.Context, resume *model.Resume, maxResumes int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls++
	if maxResumes >= 0 && len(r.rows) >= maxResumes {
		return model.ErrResumeLimitReached
	}
	if _, taken := r.rows[resume.ID]; taken {
		return model.ErrResumeAlreadyExists
	}
	stored := *resume
	r.rows[resume.ID] = &stored
	return nil
}

func (r *slotEnforcingRepo) GetByID(_ context.Context, _, resumeID string) (*model.Resume, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	row, ok := r.rows[resumeID]
	if !ok {
		return nil, model.ErrResumeNotFound
	}
	return row, nil
}

func (r *slotEnforcingRepo) List(context.Context, string, int, int, string, string) ([]*ports.ResumeWithCount, int, error) {
	return nil, 0, nil
}
func (r *slotEnforcingRepo) Update(context.Context, *model.Resume) error  { return nil }
func (r *slotEnforcingRepo) Delete(context.Context, string, string) error { return nil }
func (r *slotEnforcingRepo) stored() int                                  { r.mu.Lock(); defer r.mu.Unlock(); return len(r.rows) }
func (r *slotEnforcingRepo) attempts() int                                { r.mu.Lock(); defer r.mu.Unlock(); return r.calls }

// uploadID builds a distinct, parseable resume id per upload — the service
// refuses anything that is not a UUID before it touches storage.
func uploadID(n int) string {
	return fmt.Sprintf("3f2a6c1e-0000-4000-8000-%012d", n)
}

func uploadKey(userID string, n int) string {
	return uploadStorageKey(userID, uploadID(n))
}

/*
The plan ceiling used to be a read followed, some milliseconds later, by an
unrelated write. `CheckLimit` said "two of three used", the object was fetched
and sniffed, and only then was the row inserted — so two finalizations running
at once both passed the check and both wrote, and a three-resume plan ended up
with four. Nothing downstream noticed: the limit is only ever read again on the
next attempt.
*/
func TestResumeService_FinalizeUpload_EnforcesTheLimitAtTheWrite(t *testing.T) {
	const (
		userID   = "user-123"
		maxSlots = 3
		attempts = 8
	)

	t.Run("concurrent finalizations never exceed the plan", func(t *testing.T) {
		objects := map[string][]byte{}
		for i := 0; i < attempts; i++ {
			objects[uploadKey(userID, i)] = pdfBytes(2048)
		}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := newSlotEnforcingRepo()
		// The pre-check passes for every one of them: it is reading a count
		// that no other in-flight finalize has changed yet, which is the whole
		// reason it cannot be the enforcement.
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: maxSlots}, nil)

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			succeeded int
			refused   int
		)
		start := make(chan struct{})
		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				<-start
				_, err := svc.FinalizeUpload(context.Background(), userID, uploadID(n), nil)

				mu.Lock()
				defer mu.Unlock()
				switch {
				case err == nil:
					succeeded++
				case assert.ErrorIs(t, err, model.ErrResumeLimitReached):
					refused++
				}
			}(i)
		}
		close(start)
		wg.Wait()

		assert.Equal(t, maxSlots, succeeded, "the plan's allowance is the ceiling")
		assert.Equal(t, attempts-maxSlots, refused)
		assert.Equal(t, maxSlots, repo.stored())
		assert.Equal(t, attempts, repo.attempts(), "every attempt reached the write")
	})

	// The object belongs to nobody once its finalization is refused: the client
	// retries from a fresh presigned URL under a fresh id, so this key is never
	// named again. Left behind, it waits for a bucket lifecycle rule that may
	// not exist.
	t.Run("drops the object of a finalization that lost the race", func(t *testing.T) {
		key := uploadKey(userID, 0)
		objects := map[string][]byte{key: pdfBytes(2048)}
		s3Client, cleanup := storage.NewTestS3Client(objects)
		defer cleanup()

		repo := newSlotEnforcingRepo()
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: 0}, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, uploadID(0), nil)

		assert.ErrorIs(t, err, model.ErrResumeLimitReached)
		assert.NotContains(t, objects, key, "an unreferenced object must not be left in storage")
	})

	t.Run("passes the plan's own ceiling to the write", func(t *testing.T) {
		key := uploadKey(userID, 0)
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{key: pdfBytes(2048)})
		defer cleanup()

		var seen int
		repo := &MockResumeRepository{
			CreateFinalizedUploadFunc: func(_ context.Context, _ *model.Resume, maxResumes int) error {
				seen = maxResumes
				return nil
			},
		}
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: 5}, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, uploadID(0), nil)

		require.NoError(t, err)
		assert.Equal(t, 5, seen)
	})

	t.Run("treats an unmetered plan as unlimited", func(t *testing.T) {
		key := uploadKey(userID, 0)
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{key: pdfBytes(2048)})
		defer cleanup()

		seen := 0
		repo := &MockResumeRepository{
			CreateFinalizedUploadFunc: func(_ context.Context, _ *model.Resume, maxResumes int) error {
				seen = maxResumes
				return nil
			},
		}
		// No limit checker at all — the self-hosted configuration.
		svc := NewResumeService(repo, s3Client, nil, nil)

		_, err := svc.FinalizeUpload(context.Background(), userID, uploadID(0), nil)

		require.NoError(t, err)
		assert.Equal(t, -1, seen)
	})

	// Idempotency is what the whole fast path exists for: a retry must get the
	// resume it already created, never a limit refusal caused by the row its
	// own first attempt wrote.
	t.Run("a retry of a finalized upload is answered from the row, not the limit", func(t *testing.T) {
		key := uploadKey(userID, 0)
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{key: pdfBytes(2048)})
		defer cleanup()

		repo := newSlotEnforcingRepo()
		svc := NewResumeService(repo, s3Client, &mockLimitChecker{max: 1}, nil)

		first, err := svc.FinalizeUpload(context.Background(), userID, uploadID(0), nil)
		require.NoError(t, err)

		second, err := svc.FinalizeUpload(context.Background(), userID, uploadID(0), nil)

		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, 1, repo.stored(), "a retry must not create a second resume")
	})
}

/*
Rows the presign-era upload flow left behind.

`POST /resumes/upload-url` used to insert the row itself, inactive and with no
object behind it. Every abandoned upload left one permanently: it filled a plan
slot, and the idempotent finalize path recognised it as a completed upload and
answered "your file is stored" for a file that had never arrived.
*/
func TestResumeService_FinalizeUpload_LegacyPlaceholderRows(t *testing.T) {
	const (
		userID   = "user-123"
		resumeID = "3f2a6c1e-0000-4000-8000-00000000abcd"
	)
	objectKey := uploadStorageKey(userID, resumeID)

	placeholder := func() *model.Resume {
		key := objectKey
		created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
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

	t.Run("is not reported as a finalized upload", func(t *testing.T) {
		// Nothing was ever uploaded under that key, which is why the row is
		// still a placeholder.
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
		defer cleanup()

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return placeholder(), nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrResumeFileMissing,
			"the honest answer is that there is no file, not that the upload succeeded")
	})

	t.Run("a real upload the customer switched off is still finalized", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
		defer cleanup()

		deactivated := placeholder()
		// Switching a resume off is an update, and an update moves updated_at.
		deactivated.UpdatedAt = deactivated.CreatedAt.Add(time.Hour)

		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return deactivated, nil
			},
		}
		svc := NewResumeService(repo, s3Client, nil, nil)

		dto, err := svc.FinalizeUpload(context.Background(), userID, resumeID, nil)

		require.NoError(t, err)
		assert.Equal(t, resumeID, dto.ID)
	})
}

func TestResume_IsUnfinalizedUpload(t *testing.T) {
	created := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	key := "users/u/resumes/r.pdf"
	external := "https://example.com/cv.pdf"

	tests := []struct {
		name   string
		resume model.Resume
		want   bool
	}{
		{
			name: "a presign-era placeholder",
			resume: model.Resume{
				StorageType: model.StorageTypeS3,
				StorageKey:  &key,
				CreatedAt:   created,
				UpdatedAt:   created,
			},
			want: true,
		},
		{
			name: "a finalized upload is active",
			resume: model.Resume{
				StorageType: model.StorageTypeS3,
				StorageKey:  &key,
				IsActive:    true,
				CreatedAt:   created,
				UpdatedAt:   created,
			},
		},
		{
			name: "an upload the customer switched off has been updated",
			resume: model.Resume{
				StorageType: model.StorageTypeS3,
				StorageKey:  &key,
				CreatedAt:   created,
				UpdatedAt:   created.Add(time.Second),
			},
		},
		{
			name: "an external-URL resume is complete on creation",
			resume: model.Resume{
				StorageType: model.StorageTypeExternal,
				FileURL:     &external,
				CreatedAt:   created,
				UpdatedAt:   created,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.resume.IsUnfinalizedUpload())
		})
	}
}
