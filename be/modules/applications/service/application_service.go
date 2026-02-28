package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	commentPorts "github.com/andreypavlenko/jobber/modules/comments/ports"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ApplicationService struct {
	pool         *pgxpool.Pool
	appRepo      ports.ApplicationRepository
	stageRepo    ports.ApplicationStageRepository
	templateRepo ports.StageTemplateRepository
	jobRepo      jobPorts.JobRepository
	companyRepo  companyPorts.CompanyRepository
	resumeRepo   resumePorts.ResumeRepository
	commentRepo  commentPorts.CommentRepository
	log          *logger.Logger
}

func NewApplicationService(
	pool *pgxpool.Pool,
	appRepo ports.ApplicationRepository,
	stageRepo ports.ApplicationStageRepository,
	templateRepo ports.StageTemplateRepository,
	jobRepo jobPorts.JobRepository,
	companyRepo companyPorts.CompanyRepository,
	resumeRepo resumePorts.ResumeRepository,
	commentRepo commentPorts.CommentRepository,
	log *logger.Logger,
) *ApplicationService {
	if log == nil {
		log = &logger.Logger{Logger: zap.NewNop()}
	}
	return &ApplicationService{
		pool:         pool,
		appRepo:      appRepo,
		stageRepo:    stageRepo,
		templateRepo: templateRepo,
		jobRepo:      jobRepo,
		companyRepo:  companyRepo,
		resumeRepo:   resumeRepo,
		commentRepo:  commentRepo,
		log:          log,
	}
}

func (s *ApplicationService) Create(ctx context.Context, userID string, req *model.CreateApplicationRequest) (*model.ApplicationDTO, error) {
	appliedAt := req.AppliedAt
	if appliedAt.IsZero() {
		appliedAt = time.Now().UTC()
	}

	// Use provided name, or auto-generate from job title if empty
	name := strings.TrimSpace(req.Name)
	if name == "" {
		// Fetch job to get title for auto-naming
		if job, err := s.jobRepo.GetByID(ctx, userID, req.JobID); err == nil {
			name = job.Title
		} else {
			name = "Untitled Application"
		}
	}

	app := &model.Application{
		UserID:    userID,
		JobID:     req.JobID,
		ResumeID:  req.ResumeID,
		Name:      name,
		Status:    "active",
		AppliedAt: appliedAt,
	}

	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	// Fetch related entities for the response
	return s.buildApplicationDTO(ctx, userID, app)
}

func (s *ApplicationService) GetByID(ctx context.Context, userID, appID string) (*model.ApplicationDTO, error) {
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	// Build DTO with nested entities
	dto, err := s.buildApplicationDTO(ctx, userID, app)
	if err != nil {
		return nil, err
	}

	// Fetch and split comments
	comments, err := s.commentRepo.ListByApplication(ctx, appID)
	if err != nil {
		// Log error but don't fail the request
		s.log.Warn("failed to fetch comments for application", zap.String("application_id", appID), zap.Error(err))
	} else {
		// Split comments: application-level (stage_id == nil) vs stage-level (stage_id != nil)
		var applicationComments []*commentModel.CommentDTO
		var stageComments []*commentModel.CommentDTO

		for _, comment := range comments {
			commentDTO := comment.ToDTO()
			if comment.StageID == nil {
				applicationComments = append(applicationComments, commentDTO)
			} else {
				stageComments = append(stageComments, commentDTO)
			}
		}

		dto.ApplicationComments = applicationComments
		dto.StageComments = stageComments
	}

	return dto, nil
}

// buildApplicationDTO constructs an ApplicationDTO with all nested entities
func (s *ApplicationService) buildApplicationDTO(ctx context.Context, userID string, app *model.Application) (*model.ApplicationDTO, error) {
	// Fetch job
	job, err := s.jobRepo.GetByID(ctx, userID, app.JobID)
	if err != nil {
		s.log.Warn("failed to fetch job", zap.String("job_id", app.JobID), zap.Error(err))
		job = nil
	}

	// Fetch company if job has one
	var company *companyModel.Company
	if job != nil {
		if job.CompanyID != nil {
			company, err = s.companyRepo.GetByID(ctx, userID, *job.CompanyID)
			if err != nil {
				s.log.Warn("failed to fetch company", zap.String("company_id", *job.CompanyID), zap.Error(err))
				company = nil
			}
		} else {
			s.log.Debug("job has no company_id", zap.String("job_id", job.ID))
		}
	}

	// Fetch resume
	resume, err := s.resumeRepo.GetByID(ctx, userID, app.ResumeID)
	if err != nil {
		s.log.Warn("failed to fetch resume", zap.String("resume_id", app.ResumeID), zap.Error(err))
		resume = nil
	}

	// Get last activity
	lastActivity, err := s.appRepo.GetLastActivityAt(ctx, app.ID)
	if err != nil {
		s.log.Warn("failed to get last activity", zap.String("application_id", app.ID), zap.Error(err))
		lastActivity = app.UpdatedAt
	}

	dto := model.NewApplicationDTO(app, job, company, resume, lastActivity)

	// Resolve current stage name
	if app.CurrentStageID != nil && *app.CurrentStageID != "" {
		stage, err := s.stageRepo.GetByID(ctx, *app.CurrentStageID)
		if err != nil {
			s.log.Warn("failed to fetch current stage for DTO",
				zap.String("application_id", app.ID),
				zap.String("stage_id", *app.CurrentStageID),
				zap.Error(err))
		} else {
			tmpl, err := s.templateRepo.GetByID(ctx, userID, stage.StageTemplateID)
			if err != nil {
				s.log.Warn("failed to fetch stage template for DTO",
					zap.String("stage_template_id", stage.StageTemplateID),
					zap.Error(err))
			} else {
				dto.CurrentStageName = &tmpl.Name
			}
		}
	}

	return dto, nil
}

func (s *ApplicationService) List(ctx context.Context, userID string, sortBy, sortDir, status string, limit, offset int) ([]*model.ApplicationDTO, int, error) {
	opts := &ports.ListOptions{
		Limit:   limit,
		Offset:  offset,
		SortBy:  sortBy,
		SortDir: sortDir,
		Status:  status,
	}
	
	apps, total, err := s.appRepo.List(ctx, userID, opts)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*model.ApplicationDTO, 0, len(apps))
	for _, app := range apps {
		// Build DTO with nested entities
		dto, err := s.buildApplicationDTO(ctx, userID, app)
		if err != nil {
			s.log.Error("failed to build DTO for application", zap.String("application_id", app.ID), zap.Error(err))
			continue
		}
		dtos = append(dtos, dto)
	}
	return dtos, total, nil
}

func (s *ApplicationService) Update(ctx context.Context, userID, appID string, req *model.UpdateApplicationRequest) (*model.ApplicationDTO, error) {
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	if req.Status != nil {
		// Validate status
		validStatuses := map[string]bool{
			string(model.StatusActive):   true,
			string(model.StatusOnHold):   true,
			string(model.StatusRejected): true,
			string(model.StatusOffer):    true,
			string(model.StatusArchived): true,
		}
		if !validStatuses[*req.Status] {
			return nil, model.ErrInvalidStatus
		}
		app.Status = *req.Status
	}

	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	// Return DTO with nested entities
	return s.buildApplicationDTO(ctx, userID, app)
}

func (s *ApplicationService) Delete(ctx context.Context, userID, appID string) error {
	return s.appRepo.Delete(ctx, userID, appID)
}

// Stage management

// AddStage adds a new stage to an application following append-only semantics.
// All write operations are wrapped in a database transaction for atomicity.
func (s *ApplicationService) AddStage(ctx context.Context, userID, appID string, req *model.AddStageRequest) (*model.ApplicationStageDTO, error) {
	// Verify application belongs to user (read, outside tx)
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	// Verify stage template exists and belongs to user (read, outside tx)
	template, err := s.templateRepo.GetByID(ctx, userID, req.StageTemplateID)
	if err != nil {
		return nil, err
	}

	// Get existing stages to determine order (read, outside tx)
	existingStages, err := s.stageRepo.ListByApplication(ctx, appID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	order := len(existingStages)
	var previousStageName string

	// Begin transaction for all write operations
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Complete the current active stage (if any)
	if app.CurrentStageID != nil && *app.CurrentStageID != "" {
		currentStage, err := s.stageRepo.GetByID(ctx, *app.CurrentStageID)
		if err != nil {
			return nil, err
		}

		if currentStage.Status != "completed" {
			prevTemplate, err := s.templateRepo.GetByID(ctx, userID, currentStage.StageTemplateID)
			if err == nil {
				previousStageName = prevTemplate.Name
			}
			_, err = tx.Exec(ctx,
				`UPDATE application_stages SET status = $2, completed_at = $3 WHERE id = $1`,
				currentStage.ID, "completed", now,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to complete current stage: %w", err)
			}
		}
	}

	// Create new stage with "active" status
	newStageID := uuid.New().String()
	createdAt := time.Now().UTC()
	_, err = tx.Exec(ctx,
		`INSERT INTO application_stages (id, application_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		newStageID, appID, req.StageTemplateID, "active", order, now, nil, createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	// Update application's current stage
	_, err = tx.Exec(ctx,
		`UPDATE applications SET current_stage_id = $2, updated_at = $3 WHERE id = $1`,
		app.ID, newStageID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update application current stage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create comment if provided (outside tx — non-critical)
	if req.Comment != nil && strings.TrimSpace(*req.Comment) != "" {
		comment := &commentModel.Comment{
			UserID:        userID,
			ApplicationID: appID,
			StageID:       &newStageID,
			Content:       strings.TrimSpace(*req.Comment),
		}
		if err := s.commentRepo.Create(ctx, comment); err != nil {
			s.log.Error("failed to create comment for stage", zap.Error(err))
		}
	}

	// Log the stage change
	if previousStageName != "" {
		s.log.Info("stage changed",
			zap.String("application_id", appID),
			zap.String("previous_stage", previousStageName),
			zap.String("new_stage", template.Name),
			zap.String("user_id", userID))
	} else {
		s.log.Info("stage added",
			zap.String("application_id", appID),
			zap.String("stage", template.Name),
			zap.String("user_id", userID))
	}

	// Build the stage DTO for the response
	stage := &model.ApplicationStage{
		ID:              newStageID,
		ApplicationID:   appID,
		StageTemplateID: req.StageTemplateID,
		Status:          "active",
		Order:           order,
		StartedAt:       now,
		CreatedAt:       createdAt,
	}

	return stage.ToDTO(template.Name), nil
}

func (s *ApplicationService) CompleteStage(ctx context.Context, userID, appID, stageID string, req *model.CompleteStageRequest) (*model.ApplicationStageDTO, error) {
	// Verify application belongs to user
	_, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}

	if stage.ApplicationID != appID {
		return nil, model.ErrApplicationStageNotFound
	}

	completedAt := time.Now().UTC()
	if req.CompletedAt != nil {
		completedAt = *req.CompletedAt
	}

	stage.Status = "completed"
	stage.CompletedAt = &completedAt

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Get template for DTO
	template, err := s.templateRepo.GetByID(ctx, userID, stage.StageTemplateID)
	if err != nil {
		return nil, err
	}

	return stage.ToDTO(template.Name), nil
}

func (s *ApplicationService) ListStages(ctx context.Context, userID, appID string) ([]*model.ApplicationStageDTO, error) {
	// Verify application belongs to user
	_, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	stages, err := s.stageRepo.ListByApplication(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Get all templates to enrich DTOs (without pagination for internal use)
	templates, _, err := s.templateRepo.List(ctx, userID, 1000, 0)
	if err != nil {
		return nil, err
	}

	templateMap := make(map[string]string)
	for _, t := range templates {
		templateMap[t.ID] = t.Name
	}

	dtos := make([]*model.ApplicationStageDTO, len(stages))
	for i, stage := range stages {
		stageName := templateMap[stage.StageTemplateID]
		dtos[i] = stage.ToDTO(stageName)
	}

	return dtos, nil
}

// Stage Templates

func (s *ApplicationService) CreateStageTemplate(ctx context.Context, userID string, req *model.CreateStageTemplateRequest) (*model.StageTemplateDTO, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.ErrStageNameRequired
	}

	template := &model.StageTemplate{
		UserID: userID,
		Name:   strings.TrimSpace(req.Name),
		Order:  req.Order,
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}
	return template.ToDTO(), nil
}

func (s *ApplicationService) ListStageTemplates(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplateDTO, int, error) {
	templates, total, err := s.templateRepo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*model.StageTemplateDTO, len(templates))
	for i, t := range templates {
		dtos[i] = t.ToDTO()
	}
	return dtos, total, nil
}

func (s *ApplicationService) UpdateStageTemplate(ctx context.Context, userID, templateID string, req *model.UpdateStageTemplateRequest) (*model.StageTemplateDTO, error) {
	template, err := s.templateRepo.GetByID(ctx, userID, templateID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, model.ErrStageNameRequired
		}
		template.Name = strings.TrimSpace(*req.Name)
	}
	if req.Order != nil {
		template.Order = *req.Order
	}

	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, err
	}
	return template.ToDTO(), nil
}

func (s *ApplicationService) DeleteStageTemplate(ctx context.Context, userID, templateID string) error {
	return s.templateRepo.Delete(ctx, userID, templateID)
}

// UpdateStage updates a stage's status and other fields
func (s *ApplicationService) UpdateStage(ctx context.Context, userID, appID, stageID string, req *model.UpdateStageRequest) (*model.ApplicationStageDTO, error) {
	s.log.Debug("UpdateStage called", zap.String("user_id", userID), zap.String("application_id", appID), zap.String("stage_id", stageID))
	
	// Verify application belongs to user
	_, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		s.log.Error("failed to get application", zap.Error(err))
		return nil, err
	}

	// Get the stage
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		s.log.Error("failed to get stage", zap.Error(err))
		return nil, err
	}

	// Verify stage belongs to the application
	if stage.ApplicationID != appID {
		s.log.Error("stage does not belong to application", zap.String("stage_id", stageID), zap.String("application_id", appID))
		return nil, model.ErrApplicationStageNotFound
	}

	s.log.Debug("current stage status", zap.String("status", stage.Status), zap.Any("requested_status", req.Status))

	// Update status if provided
	if req.Status != nil {
		validStatuses := map[string]bool{
			"pending":   true,
			"active":    true,
			"completed": true,
			"skipped":   true,
			"cancelled": true,
		}
		if !validStatuses[*req.Status] {
			s.log.Error("invalid status", zap.String("status", *req.Status))
			return nil, model.ErrInvalidStatus
		}
		stage.Status = *req.Status
	}

	// Update completed_at if provided or if status is completed
	if req.CompletedAt != nil {
		stage.CompletedAt = req.CompletedAt
	} else if req.Status != nil && *req.Status == "completed" && stage.CompletedAt == nil {
		now := time.Now().UTC()
		stage.CompletedAt = &now
	}

	// Clear completed_at if status changed away from completed
	if req.Status != nil && *req.Status != "completed" && *req.Status != "skipped" && *req.Status != "cancelled" {
		stage.CompletedAt = nil
	}

	s.log.Debug("about to update stage in DB", zap.String("status", stage.Status))

	// Update in database
	if err := s.stageRepo.Update(ctx, stage); err != nil {
		s.log.Error("failed to update stage in DB", zap.Error(err))
		return nil, err
	}

	s.log.Debug("stage updated, fetching template", zap.String("stage_template_id", stage.StageTemplateID))

	// Get template for DTO
	template, err := s.templateRepo.GetByID(ctx, userID, stage.StageTemplateID)
	if err != nil {
		s.log.Error("failed to get template", zap.String("stage_template_id", stage.StageTemplateID), zap.Error(err))
		return nil, err
	}

	// Log the status change
	s.log.Info("stage status updated",
		zap.String("application_id", appID),
		zap.String("stage_id", stageID),
		zap.String("new_status", stage.Status),
		zap.String("user_id", userID))

	return stage.ToDTO(template.Name), nil
}

// DeleteStage deletes a stage from an application with validation.
// If the deleted stage is the current active stage, it recalculates current_stage_id.
// All write operations are wrapped in a database transaction for atomicity.
func (s *ApplicationService) DeleteStage(ctx context.Context, userID, appID, stageID string) error {
	// Verify application belongs to user (read, outside tx)
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return err
	}

	// Get the stage to be deleted (read, outside tx)
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}

	// Verify stage belongs to the application
	if stage.ApplicationID != appID {
		return model.ErrApplicationStageNotFound
	}

	// Check if this stage is the current active stage
	isCurrentStage := app.CurrentStageID != nil && *app.CurrentStageID == stageID

	// Begin transaction for all write operations
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// If this is the current stage, update application's current_stage_id first
	// (must happen before DELETE due to FK constraint)
	if isCurrentStage {
		// Get all stages except the one being deleted to find a replacement
		stages, err := s.stageRepo.ListByApplication(ctx, appID)
		if err != nil {
			return err
		}

		// Find the most recent active or completed stage (excluding the one being deleted)
		var newCurrentStageID *string
		for i := len(stages) - 1; i >= 0; i-- {
			if stages[i].ID == stageID {
				continue
			}
			if stages[i].Status == "active" || stages[i].Status == "completed" {
				newCurrentStageID = &stages[i].ID
				break
			}
		}

		// Update application's current stage within the transaction
		_, err = tx.Exec(ctx,
			`UPDATE applications SET current_stage_id = $2, updated_at = $3 WHERE id = $1`,
			app.ID, newCurrentStageID, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("failed to update application current stage: %w", err)
		}
	}

	// Delete the stage within the transaction
	_, err = tx.Exec(ctx,
		`DELETE FROM application_stages WHERE id = $1`,
		stageID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete stage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("stage deleted",
		zap.String("application_id", appID),
		zap.String("stage_id", stageID),
		zap.String("user_id", userID))
	return nil
}
