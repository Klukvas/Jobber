package service

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	commentPorts "github.com/andreypavlenko/jobber/modules/comments/ports"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
)

type ApplicationService struct {
	appRepo      ports.ApplicationRepository
	stageRepo    ports.ApplicationStageRepository
	templateRepo ports.StageTemplateRepository
	jobRepo      jobPorts.JobRepository
	companyRepo  companyPorts.CompanyRepository
	resumeRepo   resumePorts.ResumeRepository
	commentRepo  commentPorts.CommentRepository
}

func NewApplicationService(
	appRepo ports.ApplicationRepository,
	stageRepo ports.ApplicationStageRepository,
	templateRepo ports.StageTemplateRepository,
	jobRepo jobPorts.JobRepository,
	companyRepo companyPorts.CompanyRepository,
	resumeRepo resumePorts.ResumeRepository,
	commentRepo commentPorts.CommentRepository,
) *ApplicationService {
	return &ApplicationService{
		appRepo:      appRepo,
		stageRepo:    stageRepo,
		templateRepo: templateRepo,
		jobRepo:      jobRepo,
		companyRepo:  companyRepo,
		resumeRepo:   resumeRepo,
		commentRepo:  commentRepo,
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
		log.Printf("[WARN] Failed to fetch comments for application %s: %v", appID, err)
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
		log.Printf("[WARN] Failed to fetch job %s: %v", app.JobID, err)
		job = nil
	}

	// Fetch company if job has one
	var company *companyModel.Company
	if job != nil {
		if job.CompanyID != nil {
			company, err = s.companyRepo.GetByID(ctx, userID, *job.CompanyID)
			if err != nil {
				log.Printf("[WARN] Failed to fetch company %s: %v", *job.CompanyID, err)
				company = nil
			}
		} else {
			log.Printf("[DEBUG] Job %s has no company_id", job.ID)
		}
	}

	// Fetch resume
	resume, err := s.resumeRepo.GetByID(ctx, userID, app.ResumeID)
	if err != nil {
		log.Printf("[WARN] Failed to fetch resume %s: %v", app.ResumeID, err)
		resume = nil
	}

	// Get last activity
	lastActivity, err := s.appRepo.GetLastActivityAt(ctx, app.ID)
	if err != nil {
		log.Printf("[WARN] Failed to get last activity for %s: %v", app.ID, err)
		lastActivity = app.UpdatedAt
	}

	return model.NewApplicationDTO(app, job, company, resume, lastActivity), nil
}

func (s *ApplicationService) List(ctx context.Context, userID string, sortBy, sortDir string, limit, offset int) ([]*model.ApplicationDTO, int, error) {
	opts := &ports.ListOptions{
		Limit:   limit,
		Offset:  offset,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
	
	apps, total, err := s.appRepo.List(ctx, userID, opts)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*model.ApplicationDTO, len(apps))
	for i, app := range apps {
		// Build DTO with nested entities
		dto, err := s.buildApplicationDTO(ctx, userID, app)
		if err != nil {
			log.Printf("[ERROR] Failed to build DTO for application %s: %v", app.ID, err)
			continue
		}
		dtos[i] = dto
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

// AddStage adds a new stage to an application following append-only semantics
// This method:
// 1. Completes the current active stage (if any)
// 2. Creates a new stage with "active" status
// 3. Updates the application's current_stage_id
// Note: Ideally this should be wrapped in a database transaction for atomicity
func (s *ApplicationService) AddStage(ctx context.Context, userID, appID string, req *model.AddStageRequest) (*model.ApplicationStageDTO, error) {
	// Verify application belongs to user
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	// Verify stage template exists and belongs to user
	template, err := s.templateRepo.GetByID(ctx, userID, req.StageTemplateID)
	if err != nil {
		return nil, err
	}

	// Get existing stages to determine order
	existingStages, err := s.stageRepo.ListByApplication(ctx, appID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	order := len(existingStages)
	var previousStageID string
	var previousStageName string

	// Complete the current active stage (if any)
	// Following append-only semantics: we complete the old stage before adding new
	if app.CurrentStageID != nil && *app.CurrentStageID != "" {
		currentStage, err := s.stageRepo.GetByID(ctx, *app.CurrentStageID)
		if err != nil {
			return nil, err
		}

		// Only complete if not already completed
		if currentStage.Status != "completed" {
			previousStageID = currentStage.ID
			// Get previous stage name for logging
			prevTemplate, err := s.templateRepo.GetByID(ctx, userID, currentStage.StageTemplateID)
			if err == nil {
				previousStageName = prevTemplate.Name
			}
			currentStage.Status = "completed"
			currentStage.CompletedAt = &now
			if err := s.stageRepo.Update(ctx, currentStage); err != nil {
				return nil, err
			}
		}
	}

	// Create new stage with "active" status (it's the current stage now)
	stage := &model.ApplicationStage{
		ApplicationID:   appID,
		StageTemplateID: req.StageTemplateID,
		Status:          "active",
		Order:           order,
		StartedAt:       now,
	}

	if err := s.stageRepo.Create(ctx, stage); err != nil {
		return nil, err
	}

	// Update application's current stage
	app.CurrentStageID = &stage.ID
	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	// Create comment if provided
	if req.Comment != nil && strings.TrimSpace(*req.Comment) != "" {
		comment := &commentModel.Comment{
			UserID:        userID,
			ApplicationID: appID,
			StageID:       &stage.ID,
			Content:       strings.TrimSpace(*req.Comment),
		}
		if err := s.commentRepo.Create(ctx, comment); err != nil {
			// Log error but don't fail the stage creation
			log.Printf("[ERROR] Failed to create comment for stage: %v", err)
		}
	}

	// Log the stage change for audit trail
	if previousStageID != "" {
		log.Printf("[INFO] action=change_stage application_id=%s previous_stage=%s new_stage=%s user_id=%s",
			appID, previousStageName, template.Name, userID)
	} else {
		log.Printf("[INFO] action=add_stage application_id=%s stage=%s user_id=%s",
			appID, template.Name, userID)
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
	log.Printf("[DEBUG] UpdateStage called: userID=%s appID=%s stageID=%s", userID, appID, stageID)
	
	// Verify application belongs to user
	_, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		log.Printf("[ERROR] Failed to get application: %v", err)
		return nil, err
	}

	// Get the stage
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		log.Printf("[ERROR] Failed to get stage: %v", err)
		return nil, err
	}

	// Verify stage belongs to the application
	if stage.ApplicationID != appID {
		log.Printf("[ERROR] Stage does not belong to application")
		return nil, model.ErrApplicationStageNotFound
	}

	log.Printf("[DEBUG] Current stage status: %s, requested status: %v", stage.Status, req.Status)

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
			log.Printf("[ERROR] Invalid status: %s", *req.Status)
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

	log.Printf("[DEBUG] About to update stage in DB with status: %s", stage.Status)

	// Update in database
	if err := s.stageRepo.Update(ctx, stage); err != nil {
		log.Printf("[ERROR] Failed to update stage in DB: %v", err)
		return nil, err
	}

	log.Printf("[DEBUG] Stage updated, fetching template: %s", stage.StageTemplateID)

	// Get template for DTO
	template, err := s.templateRepo.GetByID(ctx, userID, stage.StageTemplateID)
	if err != nil {
		log.Printf("[ERROR] Failed to get template: %v", err)
		return nil, err
	}

	// Log the status change
	log.Printf("[INFO] action=update_stage_status application_id=%s stage_id=%s new_status=%s user_id=%s",
		appID, stageID, stage.Status, userID)

	return stage.ToDTO(template.Name), nil
}

// DeleteStage deletes a stage from an application with validation
// If the deleted stage is the current active stage, it updates the application's current_stage_id
func (s *ApplicationService) DeleteStage(ctx context.Context, userID, appID, stageID string) error {
	// Verify application belongs to user
	app, err := s.appRepo.GetByID(ctx, userID, appID)
	if err != nil {
		return err
	}

	// Get the stage to be deleted
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

	// Delete the stage
	if err := s.stageRepo.Delete(ctx, stageID); err != nil {
		return err
	}

	// If deleted stage was the current stage, recalculate current stage
	if isCurrentStage {
		// Get all remaining stages
		stages, err := s.stageRepo.ListByApplication(ctx, appID)
		if err != nil {
			return err
		}

		// Find the most recent active or completed stage
		var newCurrentStageID *string
		for i := len(stages) - 1; i >= 0; i-- {
			if stages[i].Status == "active" || stages[i].Status == "completed" {
				newCurrentStageID = &stages[i].ID
				break
			}
		}

		// Update application's current stage
		app.CurrentStageID = newCurrentStageID
		if err := s.appRepo.Update(ctx, app); err != nil {
			return err
		}
	}

	log.Printf("[INFO] action=delete_stage application_id=%s stage_id=%s user_id=%s", appID, stageID, userID)
	return nil
}
