package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/logger"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	commentPorts "github.com/andreypavlenko/jobber/modules/comments/ports"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
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

// resolveResumeSelection validates a resume selection and returns the pair of
// values to write, honoring mutual exclusivity and ownership: setting one kind
// (non-empty) clears the other, and an empty string clears the field. A nil
// return value means "the request said nothing about this field".
//
// A user may only attach their own uploaded resume or resume builder —
// attaching another user's resume id is rejected.
func (s *JobService) resolveResumeSelection(ctx context.Context, userID string, resumeID, resumeBuilderID *string) (*string, *string, error) {
	if resumeID != nil && *resumeID != "" && resumeBuilderID != nil && *resumeBuilderID != "" {
		return nil, nil, model.ErrBothResumeTypesSet
	}

	cleared := ""
	var resume, builder *string

	if resumeID != nil {
		if *resumeID == "" {
			resume = &cleared
		} else {
			if _, err := s.resumeRepo.GetByID(ctx, userID, *resumeID); err != nil {
				return nil, nil, model.ErrResumeNotFound
			}
			resume = resumeID
			builder = &cleared
		}
	}
	if resumeBuilderID != nil {
		if *resumeBuilderID == "" {
			builder = &cleared
		} else {
			if err := s.resumeBuilderRepo.VerifyOwnership(ctx, userID, *resumeBuilderID); err != nil {
				return nil, nil, model.ErrResumeNotFound
			}
			builder = resumeBuilderID
			resume = &cleared
		}
	}

	return resume, builder, nil
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
		if rb, err := s.resumeBuilderRepo.GetByID(ctx, userID, *job.ResumeBuilderID); err != nil {
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
	items, total, err := s.repo.List(ctx, userID, opts)
	if err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []*model.JobDTO{}
	}
	return items, total, nil
}

// Update applies a partial edit to a card.
//
// Only the fields the request carries are written, and the stored row comes
// back from the write itself. The card is never read, mutated in memory and
// written back whole: that let one PATCH overwrite a concurrent PATCH's
// untouched fields, and let a title edit carry a stale applied_at /
// current_stage_template_id over a move that had just committed.
func (s *JobService) Update(ctx context.Context, userID, jobID string, req *model.UpdateJobRequest) (*model.JobDTO, error) {
	patch := &model.JobPatch{
		Source:      req.Source,
		URL:         req.URL,
		Description: req.Description,
		IsArchived:  req.IsArchived,
	}

	if req.CompanyID != nil {
		// Validate company ownership if a non-empty company ID is provided
		if *req.CompanyID != "" {
			if _, err := s.companyRepo.GetByID(ctx, userID, *req.CompanyID); err != nil {
				return nil, model.ErrCompanyNotFound
			}
		}
		patch.CompanyID = req.CompanyID
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, model.ErrJobTitleRequired
		}
		patch.Title = &title
	}

	resumeID, resumeBuilderID, err := s.resolveResumeSelection(ctx, userID, req.ResumeID, req.ResumeBuilderID)
	if err != nil {
		return nil, err
	}
	patch.ResumeID = resumeID
	patch.ResumeBuilderID = resumeBuilderID

	// The match-score cache keys off the description, so it only needs
	// invalidating when the text actually changes. That is the one thing the
	// patch cannot tell us, so read the previous value — for the comparison
	// only; the write below is still field-scoped.
	descriptionChanged := false
	if req.Description != nil {
		previous, err := s.repo.GetByID(ctx, userID, jobID)
		if err != nil {
			return nil, err
		}
		oldDesc := ""
		if previous.Description != nil {
			oldDesc = *previous.Description
		}
		descriptionChanged = *req.Description != oldDesc
	}

	job, err := s.repo.Update(ctx, userID, jobID, patch)
	if err != nil {
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
