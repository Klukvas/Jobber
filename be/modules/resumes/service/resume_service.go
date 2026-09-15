package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/netsafe"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
)

// LimitChecker reports what the caller's plan allows.
//
// Both questions are needed and they are not the same one. `CheckLimit` gives
// the fast, friendly "is there room right now?" that a request can be refused
// on before any work is done; `ResourceLimit` gives the ceiling itself, which
// is what the write can be gated on atomically. A negative limit means
// unlimited.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
	ResourceLimit(ctx context.Context, userID, resource string) (int, error)
}

// CacheInvalidator invalidates match-score cache when source data changes.
type CacheInvalidator interface {
	InvalidateByResume(ctx context.Context, resumeID string) error
}

type ResumeService struct {
	repo             ports.ResumeRepository
	s3Client         *storage.S3Client
	s3Enabled        bool
	limitChecker     LimitChecker
	cacheInvalidator CacheInvalidator
}

func NewResumeService(repo ports.ResumeRepository, s3Client *storage.S3Client, limitChecker LimitChecker, cacheInvalidator CacheInvalidator) *ResumeService {
	return &ResumeService{
		repo:             repo,
		s3Client:         s3Client,
		s3Enabled:        s3Client != nil,
		limitChecker:     limitChecker,
		cacheInvalidator: cacheInvalidator,
	}
}

func (s *ResumeService) Create(ctx context.Context, userID string, req *model.CreateResumeRequest) (*model.ResumeDTO, error) {
	// Check subscription limit
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resumes"); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(req.Title) == "" {
		return nil, model.ErrResumeTitleRequired
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var fileURL *string
	storageType := model.StorageTypeExternal

	// If file_url is provided, use it as external storage. Validate it at write
	// time (SSRF) so a private/internal URL can never be persisted and later
	// fetched by the match-score flow.
	if req.FileURL != nil && strings.TrimSpace(*req.FileURL) != "" {
		trimmedURL := strings.TrimSpace(*req.FileURL)
		if err := netsafe.ValidateExternalURL(trimmedURL); err != nil {
			return nil, model.ErrInvalidFileURL
		}
		fileURL = &trimmedURL
	}

	resume := &model.Resume{
		UserID:      userID,
		Title:       strings.TrimSpace(req.Title),
		FileURL:     fileURL,
		StorageType: storageType,
		StorageKey:  nil,
		IsActive:    isActive,
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, err
	}
	return resume.ToDTO(), nil
}

func (s *ResumeService) GetByID(ctx context.Context, userID, resumeID string) (*model.ResumeDTO, error) {
	resume, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return nil, err
	}
	return resume.ToDTO(), nil
}

func (s *ResumeService) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*model.ResumeDTO, int, error) {
	// Validate sort parameters
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortDir == "" {
		sortDir = "desc"
	}

	resumesWithCounts, total, err := s.repo.List(ctx, userID, limit, offset, sortBy, sortDir)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*model.ResumeDTO, len(resumesWithCounts))
	for i, rwc := range resumesWithCounts {
		dtos[i] = rwc.Resume.ToDTOWithCounts(rwc.ApplicationsCount)
	}
	return dtos, total, nil
}

func (s *ResumeService) Update(ctx context.Context, userID, resumeID string, req *model.UpdateResumeRequest) (*model.ResumeDTO, error) {
	resume, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return nil, err
	}

	fileChanged := false
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, model.ErrResumeTitleRequired
		}
		resume.Title = strings.TrimSpace(*req.Title)
	}
	if req.FileURL != nil {
		fileChanged = true
		fileURL := strings.TrimSpace(*req.FileURL)
		if fileURL == "" {
			resume.FileURL = nil
		} else {
			if err := netsafe.ValidateExternalURL(fileURL); err != nil {
				return nil, model.ErrInvalidFileURL
			}
			resume.FileURL = &fileURL
		}
	}
	if req.IsActive != nil {
		resume.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, resume); err != nil {
		return nil, err
	}

	// A changed file voids everything derived from it (match-score results,
	// autofill profile) — the composite invalidator fans out to all of them.
	if fileChanged && s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByResume(ctx, resumeID); err != nil {
			log.Printf("[WARN] resume cache invalidation failed for resume=%s: %v", resumeID, err)
		}
	}

	return resume.ToDTO(), nil
}

func (s *ResumeService) Delete(ctx context.Context, userID, resumeID string) error {
	// Get resume to check storage type. Also the ownership gate: derived-cache
	// invalidation below must never run for a foreign resume id — the
	// invalidators delete by resume_id alone, so calling them pre-check would
	// let any authenticated user void another user's caches (and, for autofill
	// profiles, force a re-charged extraction).
	resume, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return err
	}

	// Invalidate resume-derived caches before deleting (FK CASCADE is a safety net)
	if s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByResume(ctx, resumeID); err != nil {
			log.Printf("[WARN] resume cache invalidation failed for resume=%s: %v", resumeID, err)
		}
	}

	// If resume uses S3 storage, delete the file from S3 first
	// This prevents orphaned files in S3 if database deletion succeeds but S3 deletion was skipped
	if resume.StorageType == model.StorageTypeS3 && resume.StorageKey != nil && s.s3Enabled {
		if err := s.s3Client.DeleteObject(ctx, *resume.StorageKey); err != nil {
			// Continue with database deletion — orphaned S3 files are less
			// harmful than orphaned DB records.
			//
			// WARN, not ERROR: the request succeeds, the customer sees the
			// resume gone, and what is left is an unreferenced object for the
			// bucket lifecycle to expire. Logged at the same level as every
			// other failed cleanup here so an ERROR in this module still means
			// something a customer noticed.
			log.Printf("[WARN] failed to delete S3 object key=%s for resume=%s: %v", *resume.StorageKey, resumeID, err)
		}
	}

	// Delete resume from database
	return s.repo.Delete(ctx, userID, resumeID)
}

// uploadStorageKey is the one place the storage path for an uploaded resume is
// built. It is always derived from the AUTHENTICATED user id, never from
// anything the caller sends, so a request can only ever address objects under
// its own prefix — that property is what lets FinalizeUpload accept a resume id
// from the client without a database lookup to prove ownership.
func uploadStorageKey(userID, resumeID string) string {
	return fmt.Sprintf("users/%s/resumes/%s.pdf", userID, resumeID)
}

// GenerateUploadURL generates a presigned URL for uploading a resume file.
//
// Deliberately creates NO database row. The row used to be inserted here, in an
// inactive state, and every abandoned upload — a closed tab, a dropped
// connection, a rejected file — left one behind: it showed up in the resume
// list as a greyed-out "Untitled Resume", it counted against the plan's resume
// limit forever, and it kept a download endpoint pointing at an object that was
// never uploaded. The resume now comes into existence in FinalizeUpload, once
// there is a verified file to attach it to.
//
// Objects uploaded but never finalized are unreferenced and invisible; expiring
// them belongs to a bucket lifecycle rule, not to request handling.
func (s *ResumeService) GenerateUploadURL(ctx context.Context, userID string, req *model.GenerateUploadURLRequest) (*model.GenerateUploadURLResponse, error) {
	// Checked here for immediate feedback, and again at finalization — this
	// call no longer consumes a slot, so it cannot be the only gate.
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resumes"); err != nil {
			return nil, err
		}
	}

	if !s.s3Enabled {
		return nil, fmt.Errorf("S3 storage is not configured")
	}

	// Validate content type
	if req.ContentType != "application/pdf" {
		return nil, fmt.Errorf("only PDF files are allowed")
	}

	resumeID := uuid.New().String()
	storageKey := uploadStorageKey(userID, resumeID)

	// Generate presigned URL (5 minutes expiry)
	expiry := 5 * time.Minute
	uploadURL, err := s.s3Client.GeneratePresignedUploadURL(ctx, storageKey, req.ContentType, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return &model.GenerateUploadURLResponse{
		ResumeID:  resumeID,
		UploadURL: uploadURL,
		ExpiresIn: int(expiry.Seconds()),
	}, nil
}

// pdfMagic is the required first bytes of any PDF file (%PDF-).
var pdfMagic = []byte{'%', 'P', 'D', 'F', '-'}

const (
	// MaxUploadedResumeBytes caps a finalized upload. The browser enforces the
	// same limit for immediate feedback; this is the one that counts.
	MaxUploadedResumeBytes = 10 * 1024 * 1024

	// contentSniffBytes is how much of the object is pulled back to check the
	// magic number — a few bytes are enough, the rest is headroom.
	contentSniffBytes = 1024
)

// defaultResumeTitle is used when the client finalizes without one.
const defaultResumeTitle = "Untitled Resume"

// FinalizeUpload verifies a freshly uploaded object and creates the resume.
//
// This is the trust boundary for resume uploads. The browser PUTs straight to
// object storage with a presigned URL, so nothing between the file picker and
// the bucket is under our control: the file extension, the declared
// Content-Type and any client-side check are all attacker-supplied.
//
// Authorization is structural rather than a lookup: the storage key is rebuilt
// from the authenticated user id, so a caller can only ever address objects in
// its own prefix — passing somebody else's resume id simply names a path that
// does not exist for them.
//
// The object is accepted only if it is non-empty, within the size cap, and
// begins with the PDF magic number. A rejected object is deleted and no row is
// written, so a spoofed file leaves nothing behind to be listed or served.
//
// The call is idempotent. A client that retries — after a timeout, a dropped
// response, or a double-tap — must get the resume it already created back,
// not an error and certainly not a deletion: the object underneath a committed
// row is live data. So the only thing this function ever deletes is an object
// it has just proved to be invalid.
func (s *ResumeService) FinalizeUpload(ctx context.Context, userID, resumeID string, req *model.FinalizeUploadRequest) (*model.ResumeDTO, error) {
	if !s.s3Enabled {
		return nil, fmt.Errorf("S3 storage is not configured")
	}

	// The id becomes part of a storage path, so it must be exactly the opaque
	// identifier we handed out — never a caller-chosen string.
	if _, err := uuid.Parse(resumeID); err != nil {
		return nil, model.ErrResumeFileMissing
	}

	storageKey := uploadStorageKey(userID, resumeID)

	// Fast idempotent path: this upload was already finalized. Answered before
	// the limit check on purpose — a retry must not be refused because the row
	// the first attempt created now fills the customer's last slot.
	//
	// `hasPlaceholder` is the other thing that lookup can find: a row the
	// presign-era upload flow wrote before there was an object. It is not a
	// finished upload, but it *is* a row that names this key — which is what
	// makes the object below unsafe to delete.
	existing, hasPlaceholder, err := s.existingUpload(ctx, userID, resumeID, storageKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing.ToDTO(), nil
	}

	title := defaultResumeTitle
	if req != nil && req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			// The object stays: it is perfectly good, this is a client sending
			// a blank title, and deleting somebody's file over that is not this
			// function's call. It is unreferenced, so the reclamation pass
			// (be/scripts/prune-orphan-resumes) is what eventually removes it.
			return nil, model.ErrResumeTitleRequired
		}
		title = trimmed
	}

	// A slot is only consumed by a resume that actually exists, so the limit is
	// enforced here — GenerateUploadURL no longer reserves anything.
	//
	// Only the ceiling is read, not a verdict. A "is there room right now?"
	// check would answer for a moment that is over by the time the row is
	// written: two finalizations racing for the last slot both pass it. The
	// number is handed to the write, which counts and inserts in one
	// transaction, and that is the single place the limit is enforced.
	maxResumes := -1
	if s.limitChecker != nil {
		limit, err := s.limitChecker.ResourceLimit(ctx, userID, "resumes")
		if err != nil {
			// Could not find out. Never a reason to delete the object: the
			// upload is fine and it is our side that failed.
			return nil, fmt.Errorf("finalize upload: read resume limit for user %s: %w", userID, err)
		}
		maxResumes = limit
	}

	prefix, size, err := s.s3Client.GetObjectPrefix(ctx, storageKey, contentSniffBytes)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			// Nothing at that key: the upload never landed (or never started).
			return nil, model.ErrResumeFileMissing
		}
		// A timeout, a throttle or a permission problem is our fault, not a
		// missing file — say so instead of blaming the customer's upload.
		return nil, fmt.Errorf("finalize upload: read object for resume %s: %w", resumeID, err)
	}

	if validationErr := validateUploadedPDF(prefix, size); validationErr != nil {
		// Safe to delete: no row references this key, and the object has just
		// been proved invalid.
		s.discardObject(ctx, storageKey)
		return nil, validationErr
	}

	resume := &model.Resume{
		ID:          resumeID,
		UserID:      userID,
		Title:       title,
		StorageType: model.StorageTypeS3,
		StorageKey:  &storageKey,
		IsActive:    true,
	}

	// Counted and written together: see ports.ResumeRepository.
	if err := s.repo.CreateFinalizedUpload(ctx, resume, maxResumes); err != nil {
		if errors.Is(err, model.ErrResumeLimitReached) {
			// The plan has no room — either it was already full, or this call
			// lost a race for the customer's last slot. Nothing points at the
			// object, so there is nothing left that could ever finalize it.
			//
			// Unless a presign-era placeholder does. That row names this key,
			// and an upload merely blocked by a full plan has to survive an
			// upgrade: deleting the file would make it unrecoverable.
			if !hasPlaceholder {
				s.discardObject(ctx, storageKey)
			}
			return nil, err
		}
		if errors.Is(err, model.ErrResumeAlreadyExists) {
			// Lost a race with a concurrent finalize of the same upload. The
			// other one won; return its row.
			winner, _, lookupErr := s.existingUpload(ctx, userID, resumeID, storageKey)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if winner != nil {
				return winner.ToDTO(), nil
			}
			// The id is taken by something that is not this upload — do not
			// touch the object, and do not claim a resume that is not ours.
			return nil, model.ErrResumeNotFound
		}
		// Deliberately no cleanup. An INSERT can commit and still report a
		// failure (a context deadline crossing the commit, a dropped
		// connection), and deleting the object here would strip the file out
		// from under a live row. Genuinely orphaned objects are the storage
		// lifecycle's problem, not this request's.
		return nil, fmt.Errorf("finalize upload: create resume %s: %w", resumeID, err)
	}

	return resume.ToDTO(), nil
}

// existingUpload reports what, if anything, already stands under this upload's
// id — scoped to the caller, so another account's row is simply not there.
//
// Two different answers, because they lead to different places:
//
//   - a finished upload, which the caller returns as-is (the idempotent path);
//   - a presign-era *placeholder*: a row written before there was an object,
//     which is not a finished upload but does name this storage key. The
//     caller needs to know, because the write can claim it, and because
//     nothing may delete a file a row still points at.
//
// A row whose storage key is not the one this call derived is neither: it
// belongs to some other upload under a colliding id, and the caller is told so.
func (s *ResumeService) existingUpload(ctx context.Context, userID, resumeID, storageKey string) (*model.Resume, bool, error) {
	existing, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		if errors.Is(err, model.ErrResumeNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("finalize upload: look up resume %s: %w", resumeID, err)
	}
	if existing == nil {
		return nil, false, nil
	}
	if existing.StorageType != model.StorageTypeS3 ||
		existing.StorageKey == nil ||
		*existing.StorageKey != storageKey {
		return nil, false, model.ErrResumeNotFound
	}
	if existing.IsUnfinalizedUpload() {
		// Written before there was an object and never completed. Reporting it
		// as a finished upload told the customer a file had been stored when
		// nothing had, and handed back a resume whose download endpoint points
		// at an empty key. So it is not returned as one — but it is reported,
		// because the write can claim it and its file must not be deleted.
		return nil, true, nil
	}
	return existing, false, nil
}

// validateUploadedPDF applies the content rules to a sniffed object prefix.
// Pure, so the rules are unit-testable without any storage.
func validateUploadedPDF(prefix []byte, size int64) error {
	if size <= 0 {
		return model.ErrResumeFileMissing
	}
	if size > MaxUploadedResumeBytes {
		return model.ErrFileTooLarge
	}
	if !bytes.HasPrefix(prefix, pdfMagic) {
		return model.ErrInvalidFileContent
	}
	return nil
}

// discardObject removes a rejected upload so nothing unverified survives in
// storage. Failures are logged, never returned: the caller is already reporting
// a rejection and must not have it masked by a cleanup error.
func (s *ResumeService) discardObject(ctx context.Context, storageKey string) {
	if err := s.s3Client.DeleteObject(ctx, storageKey); err != nil {
		log.Printf("[WARN] failed to delete rejected upload key=%s: %v", storageKey, err)
	}
}

// GenerateDownloadURL generates a presigned URL for downloading a resume file
func (s *ResumeService) GenerateDownloadURL(ctx context.Context, userID, resumeID string) (*model.DownloadURLResponse, error) {
	if !s.s3Enabled {
		return nil, fmt.Errorf("S3 storage is not configured")
	}

	// Get resume
	resume, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return nil, err
	}

	// Verify resume uses S3 storage
	if resume.StorageType != model.StorageTypeS3 {
		return nil, fmt.Errorf("resume does not use S3 storage")
	}

	if resume.StorageKey == nil {
		return nil, fmt.Errorf("resume storage key is missing")
	}

	// Generate presigned download URL (15 minutes expiry). The resume's own
	// title becomes the saved file name — the storage key is a UUID, and
	// "3f2a…-9c1e.pdf" tells the customer nothing about which resume they got.
	expiry := 15 * time.Minute
	downloadURL, err := s.s3Client.GeneratePresignedDownloadURL(ctx, *resume.StorageKey, resume.Title, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &model.DownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   int(expiry.Seconds()),
	}, nil
}
