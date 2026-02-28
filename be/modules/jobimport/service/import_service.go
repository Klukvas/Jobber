package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/jobimport/model"
)

// ImportService handles AI-powered job page parsing.
type ImportService struct {
	aiClient *ai.AnthropicClient
}

// NewImportService creates a new import service.
// aiClient may be nil if Anthropic is not configured.
func NewImportService(aiClient *ai.AnthropicClient) *ImportService {
	return &ImportService{aiClient: aiClient}
}

// ParseJobPage extracts structured job data from raw page text using AI.
func (s *ImportService) ParseJobPage(ctx context.Context, req *model.ParseJobRequest) (*model.ParseJobResponse, error) {
	if s.aiClient == nil {
		return nil, model.ErrAINotConfigured
	}

	parsed, err := s.aiClient.ParseJobPage(ctx, req.PageText, req.PageURL)
	if err != nil {
		return nil, errors.Join(model.ErrParsingFailed, fmt.Errorf("AI call: %w", err))
	}

	return &model.ParseJobResponse{
		Title:       parsed.Title,
		CompanyName: parsed.CompanyName,
		Source:      parsed.Source,
		URL:         parsed.URL,
		Description: parsed.Description,
	}, nil
}
