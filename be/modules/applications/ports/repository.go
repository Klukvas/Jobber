package ports

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
)

// ListOptions represents options for listing applications
type ListOptions struct {
	Limit   int
	Offset  int
	SortBy  string // "last_activity", "status", "company", "applied_at"
	SortDir string // "asc", "desc"
}

type ApplicationRepository interface {
	Create(ctx context.Context, app *model.Application) error
	GetByID(ctx context.Context, userID, appID string) (*model.Application, error)
	List(ctx context.Context, userID string, opts *ListOptions) ([]*model.Application, int, error)
	Update(ctx context.Context, app *model.Application) error
	Delete(ctx context.Context, userID, appID string) error
	GetLastActivityAt(ctx context.Context, appID string) (time.Time, error)
}

type StageTemplateRepository interface {
	Create(ctx context.Context, template *model.StageTemplate) error
	GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error)
	Update(ctx context.Context, template *model.StageTemplate) error
	Delete(ctx context.Context, userID, templateID string) error
}

type ApplicationStageRepository interface {
	Create(ctx context.Context, stage *model.ApplicationStage) error
	GetByID(ctx context.Context, stageID string) (*model.ApplicationStage, error)
	ListByApplication(ctx context.Context, appID string) ([]*model.ApplicationStage, error)
	Update(ctx context.Context, stage *model.ApplicationStage) error
	Delete(ctx context.Context, stageID string) error
}
