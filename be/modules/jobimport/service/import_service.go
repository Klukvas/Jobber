package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/jobimport/model"
)

// LimitChecker checks subscription limits before resource creation.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
	RecordJobParseUsage(ctx context.Context, userID string) error
}

// ImportService handles AI-powered job page parsing.
type ImportService struct {
	aiClient     *ai.AnthropicClient
	limitChecker LimitChecker
}

// NewImportService creates a new import service.
// aiClient may be nil if Anthropic is not configured.
func NewImportService(aiClient *ai.AnthropicClient, limitChecker LimitChecker) *ImportService {
	return &ImportService{aiClient: aiClient, limitChecker: limitChecker}
}

// ParseJobPage extracts structured job data from raw page text using AI.
func (s *ImportService) ParseJobPage(ctx context.Context, userID string, req *model.ParseJobRequest) (*model.ParseJobResponse, error) {
	if s.aiClient == nil {
		return nil, model.ErrAINotConfigured
	}

	// Check subscription limit for job parsing
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "job_parses"); err != nil {
			return nil, err
		}
	}

	parsed, err := s.aiClient.ParseJobPage(ctx, req.PageText, req.PageURL)
	if err != nil {
		return nil, errors.Join(model.ErrParsingFailed, fmt.Errorf("AI call: %w", err))
	}

	// Record usage after successful parse
	if s.limitChecker != nil {
		if err := s.limitChecker.RecordJobParseUsage(ctx, userID); err != nil {
			log.Printf("[ERROR] failed to record job parse usage for user=%s: %v", userID, err)
		}
	}

	return &model.ParseJobResponse{
		Title:       parsed.Title,
		CompanyName: parsed.CompanyName,
		Source:      parsed.Source,
		URL:         parsed.URL,
		Description: parsed.Description,
	}, nil
}
