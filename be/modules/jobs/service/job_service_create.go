package service

import (
	"context"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// pipelineColumnLimit bounds the "read the user's whole pipeline" queries. A
// pipeline is a handful of columns; the cap only stops a pathological account
// from loading unboundedly.
const pipelineColumnLimit = 1000

// jobPlacement is where a new card starts: the column it sits in and the
// applied_at it carries.
type jobPlacement struct {
	column    *model.StageTemplate
	appliedAt *time.Time
	// startsApplied marks a card whose starting column means the application
	// is already out (columnImpliesApplied). That is the one fact Create needs:
	// it decides both the applied_at stamp and whether the stage history has an
	// application event to record.
	startsApplied bool
}

// Create creates a new job. Without a stage or an applied date it creates a
// "saved" (wishlist) card — the Chrome extension path.
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

	placement, err := s.resolveCreatePlacement(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	job := &model.Job{
		UserID:                 userID,
		CompanyID:              optionalID(req.CompanyID),
		Title:                  strings.TrimSpace(req.Title),
		Source:                 req.Source,
		URL:                    req.URL,
		Description:            req.Description,
		CurrentStageTemplateID: &placement.column.ID,
		AppliedAt:              placement.appliedAt,
	}

	resumeID, resumeBuilderID, err := s.resolveResumeSelection(ctx, userID, req.ResumeID, req.ResumeBuilderID)
	if err != nil {
		return nil, err
	}
	job.ResumeID = optionalID(resumeID)
	job.ResumeBuilderID = optionalID(resumeBuilderID)

	// A card that starts in a column meaning "applied" has entered that column,
	// so the history has to record it — otherwise the funnel only knows the
	// card through its current position and time-to-response has nothing to
	// measure from. The row doubles as the card's single active stage. A
	// bookmarked card has nothing to record yet.
	if placement.startsApplied {
		startedAt := time.Now().UTC()
		if placement.appliedAt != nil {
			startedAt = *placement.appliedAt
		}
		stage := &model.JobStage{
			StageTemplateID: placement.column.ID,
			Status:          "active",
			Order:           0,
			StartedAt:       startedAt,
		}
		if err := s.repo.CreateWithInitialStage(ctx, job, stage); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Create(ctx, job); err != nil {
			return nil, err
		}
	}

	return s.buildJobDTO(ctx, userID, job, false)
}

// resolveCreatePlacement decides which pipeline column a new card starts in and
// what applied_at it gets.
//
//   - An explicit stage_template_id always wins — the board's "add card to this
//     column" button knows exactly where the card goes.
//   - Otherwise an applied_at ("I already applied") moves the card out of the
//     wishlist: it lands in the user's applied column, or, for a pipeline that
//     has none, in the first column that means the application is out.
//   - Otherwise the card has only been bookmarked, so it starts in the user's
//     wishlist column.
//
// A supplied date is always honored, whatever the placement turns out to be: a
// custom pipeline without an applied column must not make the toggle vanish,
// and an explicitly chosen column never overrides the date the user picked.
func (s *JobService) resolveCreatePlacement(ctx context.Context, userID string, req *model.CreateJobRequest) (*jobPlacement, error) {
	appliedAt := normalizedAppliedAt(req.AppliedAt)

	if req.StageTemplateID != nil && *req.StageTemplateID != "" {
		column, err := s.templateRepo.GetByID(ctx, userID, *req.StageTemplateID)
		if err != nil {
			return nil, err
		}
		// Whether the chosen column means "applied" is the same question Move
		// and AddStage ask of their target, answered the same way — by the
		// column's kind. One lookup, no need to load the whole pipeline.
		startsApplied := columnImpliesApplied(column)
		if appliedAt == nil && startsApplied {
			// Placing a card straight into an applied column stamps the
			// applied-at timestamp, exactly as moving it there would.
			now := time.Now().UTC()
			appliedAt = &now
		}
		return &jobPlacement{column: column, appliedAt: appliedAt, startsApplied: startsApplied}, nil
	}

	if appliedAt == nil {
		// Neither a column nor a date: the card was only bookmarked (the Chrome
		// extension, and the create form with the toggle off), so it belongs
		// wherever "not applied yet" lives on this board.
		column, err := s.bookmarkColumn(ctx, userID)
		if err != nil {
			return nil, err
		}
		placement := &jobPlacement{column: column, startsApplied: columnImpliesApplied(column)}
		if placement.startsApplied {
			// A pipeline with no wishlist at all: the only placement available
			// already means the application is out, so it is stamped like any
			// other applied placement rather than left to disagree with the
			// column the card visibly sits in.
			now := time.Now().UTC()
			placement.appliedAt = &now
		}
		return placement, nil
	}

	columns, err := s.pipeline(ctx, userID)
	if err != nil {
		return nil, err
	}

	column := appliedColumn(columns)
	return &jobPlacement{
		column:        column,
		appliedAt:     appliedAt,
		startsApplied: columnImpliesApplied(column),
	}, nil
}

// bookmarkColumn is the column a card the user has not applied to starts in.
//
// An untouched board starts with its wishlist, so the single-column read
// answers the common case outright; only a board whose first column means
// something else pays for reading the rest of the pipeline to find out where
// the wishlist went.
func (s *JobService) bookmarkColumn(ctx context.Context, userID string) (*model.StageTemplate, error) {
	first, err := s.firstStageTemplate(ctx, userID)
	if err != nil {
		return nil, err
	}
	if first.EffectiveKind() == model.StageKindSaved {
		return first, nil
	}

	columns, err := s.pipeline(ctx, userID)
	if err != nil {
		return nil, err
	}
	return savedColumn(columns), nil
}

// pipeline reads the user's whole pipeline in column order. A create has
// nowhere to put a card without one, so an empty pipeline is an error rather
// than an empty slice every placement rule would have to re-check.
func (s *JobService) pipeline(ctx context.Context, userID string) ([]*model.StageTemplate, error) {
	columns, _, err := s.templateRepo.List(ctx, userID, pipelineColumnLimit, 0)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, model.ErrStageTemplateNotFound
	}
	return columns, nil
}

// savedColumn picks the column a bookmarked card belongs in (columns must not
// be empty): the first column that means "saved, not applied yet", wherever the
// user dragged it.
//
// Position cannot answer this. Taking the pipeline's first column turned a
// bookmark into an application the moment the user pulled Applied to the front
// of their board — the card landed in Applied and got stamped with today's
// date, for a role the user had not applied to.
//
// A pipeline with no saved column keeps the old placement, its first column:
// such a board has nowhere better, and the result stays deterministic. When
// that column means the application is already out, the card is stamped
// accordingly rather than left to disagree with the column it visibly sits in.
func savedColumn(columns []*model.StageTemplate) *model.StageTemplate {
	if column := firstColumnOfKind(columns, model.StageKindSaved); column != nil {
		return column
	}
	return columns[0]
}

// appliedColumn picks the column an already-applied card belongs in, from the
// user's pipeline in column order (columns must not be empty). The preference
// chain is deterministic so the same pipeline always places the same way:
//
//  1. the first column that means "application sent" (kind applied);
//  2. failing that, the first column that means the employer already came back
//     (response, interview, offer) — a pipeline can legitimately start at
//     "Screening";
//  3. failing that, the first unclassified column after the initial one, i.e.
//     one step out of the wishlist;
//  4. failing that, the first column that is not a rejection — including the
//     initial one. The date is what matters and there is nowhere better to put
//     the card, but another wishlist column still beats "Rejected".
//
// "I applied" is not "I was rejected", so a rejected column is chosen only when
// the pipeline offers literally nothing else: an all-rejected pipeline falls
// through to its first column. That is an honest dead end rather than a hidden
// one — this function places cards in columns the user owns and never invents
// one, and failing the create instead would leave such an account unable to add
// a card at all. The supplied date is stored either way.
func appliedColumn(columns []*model.StageTemplate) *model.StageTemplate {
	if column := firstColumnOfKind(columns, model.StageKindApplied); column != nil {
		return column
	}
	if column := firstColumnOfKind(columns,
		model.StageKindResponse, model.StageKindInterview, model.StageKindOffer); column != nil {
		return column
	}
	if column := firstColumnOfKind(columns[1:], model.StageKindOther); column != nil {
		return column
	}
	for _, column := range columns {
		if column.EffectiveKind() != model.StageKindRejected {
			return column
		}
	}
	return columns[0]
}

// firstColumnOfKind returns the first column whose kind is one of kinds, or nil
// when the pipeline has none.
func firstColumnOfKind(columns []*model.StageTemplate, kinds ...model.StageKind) *model.StageTemplate {
	for _, column := range columns {
		for _, kind := range kinds {
			if column.EffectiveKind() == kind {
				return column
			}
		}
	}
	return nil
}

// normalizedAppliedAt keeps stored timestamps in UTC, like every other
// timestamp this service writes. A zero value is treated as "not supplied" so a
// client sending `"applied_at": "0001-01-01T00:00:00Z"` cannot date a card to
// year one.
func normalizedAppliedAt(appliedAt *time.Time) *time.Time {
	if appliedAt == nil || appliedAt.IsZero() {
		return nil
	}
	utc := appliedAt.UTC()
	return &utc
}

// optionalID maps the API's "empty string means no value" convention onto a
// nullable model field.
func optionalID(id *string) *string {
	if id == nil || *id == "" {
		return nil
	}
	return id
}
