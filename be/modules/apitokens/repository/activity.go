package repository

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
)

func (r *APITokenRepository) ListActivity(ctx context.Context, userID string, limit int) ([]*model.ActivityDTO, error) {
	rows, err := r.pool.Query(ctx, `SELECT a.id, a.tool, a.ok, a.error_code, a.duration_ms,
 a.created_at, a.api_token_id, t.name
 FROM mcp_audit_log a LEFT JOIN api_tokens t ON t.id = a.api_token_id AND t.user_id = a.user_id
 WHERE a.user_id = $1 ORDER BY a.created_at DESC, a.id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list mcp activity: %w", err)
	}
	defer rows.Close()
	activity := make([]*model.ActivityDTO, 0)
	for rows.Next() {
		var item model.ActivityDTO
		if err := rows.Scan(&item.ID, &item.Tool, &item.OK, &item.ErrorCode, &item.DurationMS,
			&item.CreatedAt, &item.APITokenID, &item.TokenName); err != nil {
			return nil, fmt.Errorf("scan mcp activity: %w", err)
		}
		activity = append(activity, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mcp activity: %w", err)
	}
	return activity, nil
}
