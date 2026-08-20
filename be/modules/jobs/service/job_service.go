package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/logger"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	commentPorts "github.com/andreypavlenko/jobber/modules/comments/ports"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// txBeginner is the minimal transaction-starting surface JobService needs from
// its database pool. *pgxpool.Pool satisfies it in production; tests inject a
// pgxmock pool. Depending on this narrow interface (rather than the concrete
// *pgxpool.Pool) keeps the transactional methods unit-testable.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// LimitChecker checks subscription limits before resource creation.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
}

// CacheInvalidator invalidates match-score cache when source data changes.
type CacheInvalidator interface {
	InvalidateByJob(ctx context.Context, jobID string) error
}

// JobService handles job business logic: the job card itself and its
// application pipeline (status, stages, comments enrichment).
type JobService struct {
	pool              txBeginner
	repo              ports.JobRepository
	stageRepo         ports.JobStageRepository
	templateRepo      ports.StageTemplateRepository
	companyRepo       companyPorts.CompanyRepository
	resumeRepo        resumePorts.ResumeRepository
	resumeBuilderRepo rbPorts.ResumeBuilderRepository
	commentRepo       commentPorts.CommentRepository
	log               *logger.Logger
	limitChecker      LimitChecker
	cacheInvalidator  CacheInvalidator
}

// NewJobService creates a new job service
func NewJobService(
	pool txBeginner,
	repo ports.JobRepository,
	stageRepo ports.JobStageRepository,
	templateRepo ports.StageTemplateRepository,
	companyRepo companyPorts.CompanyRepository,
	resumeRepo resumePorts.ResumeRepository,
	resumeBuilderRepo rbPorts.ResumeBuilderRepository,
	commentRepo commentPorts.CommentRepository,
	log *logger.Logger,
	limitChecker LimitChecker,
	cacheInvalidator CacheInvalidator,
) *JobService {
	if log == nil {
		log = &logger.Logger{Logger: zap.NewNop()}
	}
	return &JobService{
		pool:              pool,
		repo:              repo,
		stageRepo:         stageRepo,
		templateRepo:      templateRepo,
		companyRepo:       companyRepo,
		resumeRepo:        resumeRepo,
		resumeBuilderRepo: resumeBuilderRepo,
		commentRepo:       commentRepo,
		log:               log,
		limitChecker:      limitChecker,
		cacheInvalidator:  cacheInvalidator,
	}
}

// resolveResumeSelection assigns resume fields honoring mutual exclusivity and
// verifying ownership: setting one kind (non-empty) clears the other; empty
// string clears the field. A user may only attach their own uploaded resume or
// resume builder — attaching another user's resume id is rejected.
func (s *JobService) resolveResumeSelection(ctx context.Context, userID string, job *model.Job, resumeID, resumeBuilderID *string) error {
	if resumeID != nil && *resumeID != "" && resumeBuilderID != nil && *resumeBuilderID != "" {
		return model.ErrBothResumeTypesSet
	}
	if resumeID != nil {
		if *resumeID == "" {
			job.ResumeID = nil
		} else {
			if _, err := s.resumeRepo.GetByID(ctx, userID, *resumeID); err != nil {
				return model.ErrResumeNotFound
			}
			id := *resumeID
			job.ResumeID = &id
			job.ResumeBuilderID = nil
		}
	}
	if resumeBuilderID != nil {
		if *resumeBuilderID == "" {
			job.ResumeBuilderID = nil
		} else {
			if err := s.resumeBuilderRepo.VerifyOwnership(ctx, userID, *resumeBuilderID); err != nil {
				return model.ErrResumeNotFound
			}
			id := *resumeBuilderID
			job.ResumeBuilderID = &id
			job.ResumeID = nil
		}
	}
	return nil
}

// firstStageTemplate returns the user's first pipeline column (lowest order) —
// the default placement for a new card.
func (s *JobService) firstStageTemplate(ctx context.Context, userID string) (*model.StageTemplate, error) {
	templates, _, err := s.templateRepo.List(ctx, userID, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, model.ErrStageTemplateNotFound
	}
	return templates[0], nil
}

// Create creates a new job. Without status/resume fields it creates a "saved"
// (wishlist) card — the Chrome extension path.
func (s *JobService) Create(ctx context.Context, userID string, req *model.CreateJobRequest) (*model.JobDTO, error) {
	// Check subscription limit
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "jobs"); err != nil {
			return nil, err
		}
	}

	// Validate
	if strings.TrimSpace(req.Title) == "" {
		return nil, model.ErrJobTitleRequired
	}

	// Validate company ownership if provided
	if req.CompanyID != nil && *req.CompanyID != "" {
		if _, err := s.companyRepo.GetByID(ctx, userID, *req.CompanyID); err != nil {
			return nil, model.ErrCompanyNotFound
		}
	}

	// Determine which pipeline column to place the card in: an explicit one, or
	// the user's first column (wishlist).
	var placement *model.StageTemplate
	if req.StageTemplateID != nil && *req.StageTemplateID != "" {
		p, err := s.templateRepo.GetByID(ctx, userID, *req.StageTemplateID)
		if err != nil {
			return nil, err
		}
		placement = p
	} else {
		p, err := s.firstStageTemplate(ctx, userID)
		if err != nil {
			return nil, err
		}
		placement = p
	}

	job := &model.Job{
		UserID:                 userID,
		CompanyID:              req.CompanyID,
		Title:                  strings.TrimSpace(req.Title),
		Source:                 req.Source,
		URL:                    req.URL,
		Description:            req.Description,
		CurrentStageTemplateID: &placement.ID,
	}
	// Placing directly past the first column stamps the applied-at timestamp.
	if placement.Order > 0 {
		now := time.Now().UTC()
		job.AppliedAt = &now
	}

	if err := s.resolveResumeSelection(ctx, userID, job, req.ResumeID, req.ResumeBuilderID); err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	return s.buildJobDTO(ctx, userID, job, false)
}

// GetByID retrieves a job by ID with nested entities and comments
func (s *JobService) GetByID(ctx context.Context, userID, jobID string) (*model.JobDTO, error) {
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	return s.buildJobDTO(ctx, userID, job, true)
}

// buildJobDTO constructs a JobDTO with nested entities. When withComments is
// true the job's comments are fetched and split into job-level and stage-level.
func (s *JobService) buildJobDTO(ctx context.Context, userID string, job *model.Job, withComments bool) (*model.JobDTO, error) {
	dto := job.ToDTO()

	// Company name
	if job.CompanyID != nil && *job.CompanyID != "" {
		if company, err := s.companyRepo.GetByID(ctx, userID, *job.CompanyID); err != nil {
			s.log.Warn("failed to fetch company", zap.String("company_id", *job.CompanyID), zap.Error(err))
		} else {
			dto.CompanyName = &company.Name
		}
	}

	// Resume (uploaded or builder — mutually exclusive)
	if job.ResumeID != nil {
		if r, err := s.resumeRepo.GetByID(ctx, userID, *job.ResumeID); err != nil {
			s.log.Warn("failed to fetch resume", zap.String("resume_id", *job.ResumeID), zap.Error(err))
		} else {
			dto.Resume = &model.ResumeNestedDTO{ID: r.ID, Name: r.Title, Type: "uploaded"}
		}
	} else if job.ResumeBuilderID != nil {
		if rb, err := s.resumeBuilderRepo.GetByID(ctx, *job.ResumeBuilderID); err != nil {
			s.log.Warn("failed to fetch resume builder", zap.String("resume_builder_id", *job.ResumeBuilderID), zap.Error(err))
		} else {
			dto.Resume = &model.ResumeNestedDTO{ID: rb.ID, Name: rb.Title, Type: "builder"}
		}
	}

	// Last activity
	if lastActivity, err := s.repo.GetLastActivityAt(ctx, userID, job.ID); err != nil {
		s.log.Warn("failed to get last activity", zap.String("job_id", job.ID), zap.Error(err))
		dto.LastActivityAt = job.UpdatedAt
	} else {
		dto.LastActivityAt = lastActivity
	}

	// Current column (stage) name — the single-axis position
	if job.CurrentStageTemplateID != nil && *job.CurrentStageTemplateID != "" {
		if tmpl, err := s.templateRepo.GetByID(ctx, userID, *job.CurrentStageTemplateID); err != nil {
			s.log.Warn("failed to fetch current stage template for DTO",
				zap.String("job_id", job.ID),
				zap.String("stage_template_id", *job.CurrentStageTemplateID),
				zap.Error(err))
		} else {
			dto.CurrentStageName = &tmpl.Name
		}
	}

	// Comments split: job-level (stage_id == nil) vs stage-level.
	// userID is passed so ListByJob scopes the query to the owner's jobs.
	if withComments {
		comments, err := s.commentRepo.ListByJob(ctx, job.ID, userID)
		if err != nil {
			s.log.Warn("failed to fetch comments for job", zap.String("job_id", job.ID), zap.Error(err))
		} else {
			var jobComments []*commentModel.CommentDTO
			var stageComments []*commentModel.CommentDTO
			for _, comment := range comments {
				commentDTO := comment.ToDTO()
				if comment.StageID == nil {
					jobComments = append(jobComments, commentDTO)
				} else {
					stageComments = append(stageComments, commentDTO)
				}
			}
			dto.JobComments = jobComments
			dto.StageComments = stageComments
		}
	}

	return dto, nil
}

// List retrieves enriched jobs for a user with pagination, filtering, and sorting
func (s *JobService) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
	return s.repo.List(ctx, userID, opts)
}

// Update updates a job, handling pipeline status transitions and resume selection
func (s *JobService) Update(ctx context.Context, userID, jobID string, req *model.UpdateJobRequest) (*model.JobDTO, error) {
	// Get existing job
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.CompanyID != nil {
		// Validate company ownership if a non-empty company ID is provided
		if *req.CompanyID != "" {
			if _, err := s.companyRepo.GetByID(ctx, userID, *req.CompanyID); err != nil {
				return nil, model.ErrCompanyNotFound
			}
		}
		job.CompanyID = req.CompanyID
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, model.ErrJobTitleRequired
		}
		job.Title = strings.TrimSpace(*req.Title)
	}
	if req.Source != nil {
		job.Source = req.Source
	}
	if req.URL != nil {
		job.URL = req.URL
	}
	descriptionChanged := false
	if req.Description != nil {
		oldDesc := ""
		if job.Description != nil {
			oldDesc = *job.Description
		}
		descriptionChanged = *req.Description != oldDesc
		job.Description = req.Description
	}

	if err := s.resolveResumeSelection(ctx, userID, job, req.ResumeID, req.ResumeBuilderID); err != nil {
		return nil, err
	}

	if req.IsArchived != nil {
		job.IsArchived = *req.IsArchived
	}

	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}

	// Invalidate match-score cache when description changes
	if descriptionChanged && s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByJob(ctx, jobID); err != nil {
			s.log.Warn("match score cache invalidation failed", zap.String("job_id", jobID), zap.Error(err))
		}
	}

	return s.buildJobDTO(ctx, userID, job, false)
}

// ToggleFavorite toggles the favorite status of a job
func (s *JobService) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	return s.repo.ToggleFavorite(ctx, userID, jobID)
}

// Delete deletes a job
func (s *JobService) Delete(ctx context.Context, userID, jobID string) error {
	// Invalidate match-score cache before deleting (FK CASCADE is a safety net)
	if s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByJob(ctx, jobID); err != nil {
			s.log.Warn("match score cache invalidation failed", zap.String("job_id", jobID), zap.Error(err))
		}
	}

	return s.repo.Delete(ctx, userID, jobID)
}

// Stage management

// AddStage adds a new stage to a job following append-only semantics.
// All write operations are wrapped in a database transaction for atomicity.
func (s *JobService) AddStage(ctx context.Context, userID, jobID string, req *model.AddStageRequest) (*model.JobStageDTO, error) {
	// Verify job belongs to user (read, outside tx)
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// Verify stage template exists and belongs to user (read, outside tx)
	template, err := s.templateRepo.GetByID(ctx, userID, req.StageTemplateID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var previousStageName string

	// Begin transaction for all write operations
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Lock the job row so concurrent AddStage/DeleteStage on the same job
	// serialize — otherwise two concurrent adds could compute the same order.
	lockTag, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	if lockTag.RowsAffected() == 0 {
		return nil, model.ErrJobNotFound
	}

	// Next order = max existing + 1 (taken under the lock). Using MAX rather
	// than COUNT keeps orders unique after a middle stage is deleted.
	var order int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX("order"), -1) + 1 FROM job_stages WHERE job_id = $1`, jobID).Scan(&order); err != nil {
		return nil, fmt.Errorf("failed to compute stage order: %w", err)
	}

	// Complete the current active stage (if any)
	if job.CurrentStageID != nil && *job.CurrentStageID != "" {
		currentStage, err := s.stageRepo.GetByID(ctx, *job.CurrentStageID, job.ID)
		if err != nil {
			return nil, err
		}

		if currentStage.Status != "completed" {
			prevTemplate, err := s.templateRepo.GetByID(ctx, userID, currentStage.StageTemplateID)
			if err == nil {
				previousStageName = prevTemplate.Name
			}
			_, err = tx.Exec(ctx,
				`UPDATE job_stages SET status = $2, completed_at = $3 WHERE id = $1`,
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
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		newStageID, jobID, req.StageTemplateID, "active", order, now, nil, createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	// Update the job's current column + history pointer. Stamp applied_at the
	// first time a card advances past its first column.
	appliedAt := job.AppliedAt
	if appliedAt == nil && template.Order > 0 {
		appliedAt = &now
	}
	_, err = tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
		job.ID, newStageID, template.ID, appliedAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update job current stage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create comment if provided (outside tx — non-critical)
	if req.Comment != nil && strings.TrimSpace(*req.Comment) != "" {
		comment := &commentModel.Comment{
			UserID:  userID,
			JobID:   jobID,
			StageID: &newStageID,
			Content: strings.TrimSpace(*req.Comment),
		}
		if err := s.commentRepo.Create(ctx, comment); err != nil {
			s.log.Error("failed to create comment for stage", zap.Error(err))
		}
	}

	// Log the stage change
	if previousStageName != "" {
		s.log.Info("stage changed",
			zap.String("job_id", jobID),
			zap.String("previous_stage", previousStageName),
			zap.String("new_stage", template.Name),
			zap.String("user_id", userID))
	} else {
		s.log.Info("stage added",
			zap.String("job_id", jobID),
			zap.String("stage", template.Name),
			zap.String("user_id", userID))
	}

	// Build the stage DTO for the response
	stage := &model.JobStage{
		ID:              newStageID,
		JobID:           jobID,
		StageTemplateID: req.StageTemplateID,
		Status:          "active",
		Order:           order,
		StartedAt:       now,
		CreatedAt:       createdAt,
	}

	return stage.ToDTO(template.Name), nil
}

// ListStages lists all stages of a job
func (s *JobService) ListStages(ctx context.Context, userID, jobID string) ([]*model.JobStageDTO, error) {
	// Verify job belongs to user
	_, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	stages, err := s.stageRepo.ListByJob(ctx, jobID)
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

	dtos := make([]*model.JobStageDTO, len(stages))
	for i, stage := range stages {
		stageName := templateMap[stage.StageTemplateID]
		dtos[i] = stage.ToDTO(stageName)
	}

	return dtos, nil
}

// UpdateStage updates a stage's status and other fields
func (s *JobService) UpdateStage(ctx context.Context, userID, jobID, stageID string, req *model.UpdateStageRequest) (*model.JobStageDTO, error) {
	// Verify job belongs to user
	_, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Serialize with AddStage/DeleteStage/Move (which all lock the job row
	// first) so a concurrent stage add can't clobber completed_at mid-write.
	lockTag, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	if lockTag.RowsAffected() == 0 {
		return nil, model.ErrJobNotFound
	}

	// Get the stage (read is safe under the job lock — no concurrent stage
	// writer for this job can commit while we hold it)
	stage, err := s.stageRepo.GetByID(ctx, stageID, jobID)
	if err != nil {
		return nil, err
	}

	// Verify stage belongs to the job
	if stage.JobID != jobID {
		return nil, model.ErrJobStageNotFound
	}

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

	// Clear completed_at if status changed away from completion states
	if req.Status != nil && *req.Status != "completed" && *req.Status != "skipped" && *req.Status != "cancelled" {
		stage.CompletedAt = nil
	}

	// Update within the transaction (under the job lock taken above)
	res, err := tx.Exec(ctx,
		`UPDATE job_stages SET status = $2, completed_at = $3 WHERE id = $1`,
		stage.ID, stage.Status, stage.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, model.ErrJobStageNotFound
	}

	// Get template for DTO
	template, err := s.templateRepo.GetByID(ctx, userID, stage.StageTemplateID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("stage status updated",
		zap.String("job_id", jobID),
		zap.String("stage_id", stageID),
		zap.String("new_status", stage.Status),
		zap.String("user_id", userID))

	return stage.ToDTO(template.Name), nil
}

// DeleteStage deletes a stage from a job with validation.
// If the deleted stage is the current active stage, it recalculates current_stage_id.
// All write operations are wrapped in a database transaction for atomicity.
func (s *JobService) DeleteStage(ctx context.Context, userID, jobID, stageID string) error {
	// Verify job belongs to user (read, outside tx)
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return err
	}

	// Get the stage to be deleted (read, outside tx)
	stage, err := s.stageRepo.GetByID(ctx, stageID, jobID)
	if err != nil {
		return err
	}

	// Verify stage belongs to the job
	if stage.JobID != jobID {
		return model.ErrJobStageNotFound
	}

	// Check if this stage is the current active stage
	isCurrentStage := job.CurrentStageID != nil && *job.CurrentStageID == stageID

	// Begin transaction for all write operations
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Lock the job row so a concurrent AddStage cannot commit between our read
	// of current_stage_id and the recalculation below (which would clobber it).
	if _, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, jobID); err != nil {
		return fmt.Errorf("failed to lock job: %w", err)
	}

	// If this is the current stage, update job's current_stage_id first
	// (must happen before DELETE due to FK constraint)
	if isCurrentStage {
		// Get all stages except the one being deleted to find a replacement
		stages, err := s.stageRepo.ListByJob(ctx, jobID)
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

		// Update job's current stage within the transaction
		_, err = tx.Exec(ctx,
			`UPDATE jobs SET current_stage_id = $2, updated_at = $3 WHERE id = $1`,
			job.ID, newCurrentStageID, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("failed to update job current stage: %w", err)
		}
	}

	// Delete the stage within the transaction
	_, err = tx.Exec(ctx,
		`DELETE FROM job_stages WHERE id = $1`,
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
		zap.String("job_id", jobID),
		zap.String("stage_id", stageID),
		zap.String("user_id", userID))
	return nil
}

// Stage Templates

func (s *JobService) CreateStageTemplate(ctx context.Context, userID string, req *model.CreateStageTemplateRequest) (*model.StageTemplateDTO, error) {
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

// ReorderStageTemplates sets a new order for the user's pipeline columns.
func (s *JobService) ReorderStageTemplates(ctx context.Context, userID string, stageIDs []string) error {
	return s.templateRepo.Reorder(ctx, userID, stageIDs)
}

func (s *JobService) ListStageTemplates(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplateDTO, int, error) {
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

func (s *JobService) UpdateStageTemplate(ctx context.Context, userID, templateID string, req *model.UpdateStageTemplateRequest) (*model.StageTemplateDTO, error) {
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

func (s *JobService) DeleteStageTemplate(ctx context.Context, userID, templateID string) error {
	return s.templateRepo.Delete(ctx, userID, templateID)
}
