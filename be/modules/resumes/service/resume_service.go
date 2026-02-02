package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
)

type ResumeService struct {
	repo      ports.ResumeRepository
	s3Client  *storage.S3Client
	s3Enabled bool
}

func NewResumeService(repo ports.ResumeRepository, s3Client *storage.S3Client) *ResumeService {
	return &ResumeService{
		repo:      repo,
		s3Client:  s3Client,
		s3Enabled: s3Client != nil,
	}
}

func (s *ResumeService) Create(ctx context.Context, userID string, req *model.CreateResumeRequest) (*model.ResumeDTO, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, model.ErrResumeTitleRequired
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var fileURL *string
	storageType := model.StorageTypeExternal

	// If file_url is provided, use it as external storage
	if req.FileURL != nil && strings.TrimSpace(*req.FileURL) != "" {
		trimmedURL := strings.TrimSpace(*req.FileURL)
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

	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, model.ErrResumeTitleRequired
		}
		resume.Title = strings.TrimSpace(*req.Title)
	}
	if req.FileURL != nil {
		fileURL := strings.TrimSpace(*req.FileURL)
		if fileURL == "" {
			resume.FileURL = nil
		} else {
			resume.FileURL = &fileURL
		}
	}
	if req.IsActive != nil {
		resume.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, resume); err != nil {
		return nil, err
	}
	return resume.ToDTO(), nil
}

func (s *ResumeService) Delete(ctx context.Context, userID, resumeID string) error {
	// Get resume to check storage type
	resume, err := s.repo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return err
	}

	// If resume uses S3 storage, delete the file from S3 first
	// This prevents orphaned files in S3 if database deletion succeeds but S3 deletion was skipped
	if resume.StorageType == model.StorageTypeS3 && resume.StorageKey != nil && s.s3Enabled {
		// Attempt to delete S3 object
		if err := s.s3Client.DeleteObject(ctx, *resume.StorageKey); err != nil {
			// Log the error but continue with database deletion
			// Rationale: Better to have orphaned S3 file than orphaned DB record
			// Orphaned S3 files can be cleaned up via bucket lifecycle policies
			// TODO: Add structured logging
			// logger.Error("Failed to delete S3 object", 
			//     zap.String("resume_id", resumeID), 
			//     zap.String("storage_key", *resume.StorageKey), 
			//     zap.Error(err))
			
			// For now, just print to stderr (will appear in server logs)
			fmt.Printf("Warning: Failed to delete S3 object for resume %s: %v\n", resumeID, err)
		}
	}

	// Delete resume from database
	return s.repo.Delete(ctx, userID, resumeID)
}

// GenerateUploadURL generates a presigned URL for uploading a resume file
func (s *ResumeService) GenerateUploadURL(ctx context.Context, userID string, req *model.GenerateUploadURLRequest) (*model.GenerateUploadURLResponse, error) {
	if !s.s3Enabled {
		return nil, fmt.Errorf("S3 storage is not configured")
	}

	// Validate content type
	if req.ContentType != "application/pdf" {
		return nil, fmt.Errorf("only PDF files are allowed")
	}

	// Generate resume ID
	resumeID := uuid.New().String()

	// Generate S3 key: users/{user_id}/resumes/{resume_id}.pdf
	storageKey := fmt.Sprintf("users/%s/resumes/%s.pdf", userID, resumeID)

	// Generate presigned URL (5 minutes expiry)
	expiry := 5 * time.Minute
	uploadURL, err := s.s3Client.GeneratePresignedUploadURL(ctx, storageKey, req.ContentType, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Create resume record with S3 storage type
	resume := &model.Resume{
		ID:          resumeID,
		UserID:      userID,
		Title:       "Untitled Resume", // Default title, user can update later
		FileURL:     nil,
		StorageType: model.StorageTypeS3,
		StorageKey:  &storageKey,
		IsActive:    false, // Default to inactive until file is uploaded
	}

	if err := s.repo.Create(ctx, resume); err != nil {
		return nil, fmt.Errorf("failed to create resume record: %w", err)
	}

	return &model.GenerateUploadURLResponse{
		ResumeID:  resumeID,
		UploadURL: uploadURL,
		ExpiresIn: int(expiry.Seconds()),
	}, nil
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

	// Generate presigned download URL (15 minutes expiry)
	expiry := 15 * time.Minute
	downloadURL, err := s.s3Client.GeneratePresignedDownloadURL(ctx, *resume.StorageKey, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	return &model.DownloadURLResponse{
		DownloadURL: downloadURL,
		ExpiresIn:   int(expiry.Seconds()),
	}, nil
}
