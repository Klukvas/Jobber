package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
)

func (r *SubscriptionRepository) ManualAccess(ctx context.Context, user string) (*model.ManualAccess, error) {
	var grant model.ManualAccess
	err := r.pool.QueryRow(ctx, `SELECT g.id::text,g.period_start,g.period_end FROM credit_grants g JOIN manual_access_grants m ON m.grant_id=g.id WHERE g.user_id=$1 AND g.kind='manual_pro' AND g.revoked_at IS NULL AND g.period_start<=now() AND g.period_end>now() ORDER BY g.period_end DESC LIMIT 1`, user).Scan(&grant.ID, &grant.PeriodStart, &grant.PeriodEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manual access: %w", err)
	}
	return &grant, nil
}
