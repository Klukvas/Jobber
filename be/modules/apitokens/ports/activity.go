package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
)

type ActivityRepository interface {
	ListActivity(ctx context.Context, userID string, limit int) ([]*model.ActivityDTO, error)
}
