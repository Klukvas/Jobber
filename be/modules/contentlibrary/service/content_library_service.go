package service

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/andreypavlenko/jobber/modules/contentlibrary/ports"
)

// ContentLibraryService handles content library business logic.
type ContentLibraryService struct {
	repo ports.ContentLibraryRepository
}

// NewContentLibraryService creates a new ContentLibraryService.
func NewContentLibraryService(repo ports.ContentLibraryRepository) *ContentLibraryService {
	return &ContentLibraryService{repo: repo}
}

// Create creates a new content library entry.
func (s *ContentLibraryService) Create(ctx context.Context, userID string, req *model.CreateContentLibraryRequest) (*model.ContentLibraryEntryDTO, error) {
	category := req.Category
	if category == "" {
		category = "general"
	}

	entry := &model.ContentLibraryEntry{
		UserID:   userID,
		Title:    req.Title,
		Content:  req.Content,
		Category: category,
	}

	created, err := s.repo.Create(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("failed to create content library entry: %w", err)
	}

	return created.ToDTO(), nil
}

// List returns all content library entries for a user.
func (s *ContentLibraryService) List(ctx context.Context, userID string) ([]*model.ContentLibraryEntryDTO, error) {
	entries, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*model.ContentLibraryEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, e.ToDTO())
	}

	return dtos, nil
}

// Update updates a content library entry.
func (s *ContentLibraryService) Update(ctx context.Context, userID, id string, req *model.UpdateContentLibraryRequest) (*model.ContentLibraryEntryDTO, error) {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if entry.UserID != userID {
		return nil, fmt.Errorf("not authorized")
	}

	if req.Title != nil {
		entry.Title = *req.Title
	}
	if req.Content != nil {
		entry.Content = *req.Content
	}
	if req.Category != nil {
		entry.Category = *req.Category
	}

	updated, err := s.repo.Update(ctx, entry)
	if err != nil {
		return nil, err
	}

	return updated.ToDTO(), nil
}

// Delete deletes a content library entry.
func (s *ContentLibraryService) Delete(ctx context.Context, userID, id string) error {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if entry.UserID != userID {
		return fmt.Errorf("not authorized")
	}

	return s.repo.Delete(ctx, id)
}
