package repository

import (
	"context"
	"errors"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SubscriptionRepository implements ports.SubscriptionRepository with PostgreSQL.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

// GetByUserID retrieves a subscription by user ID.
func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	query := `
		SELECT id, user_id, paddle_subscription_id, paddle_customer_id,
		       status, plan, current_period_start, current_period_end,
		       cancel_at, created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1`

	var sub model.Subscription
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PaddleSubscriptionID, &sub.PaddleCustomerID,
		&sub.Status, &sub.Plan, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// GetByPaddleSubscriptionID retrieves a subscription by Paddle subscription ID.
func (r *SubscriptionRepository) GetByPaddleSubscriptionID(ctx context.Context, paddleSubID string) (*model.Subscription, error) {
	query := `
		SELECT id, user_id, paddle_subscription_id, paddle_customer_id,
		       status, plan, current_period_start, current_period_end,
		       cancel_at, created_at, updated_at
		FROM subscriptions
		WHERE paddle_subscription_id = $1`

	var sub model.Subscription
	err := r.pool.QueryRow(ctx, query, paddleSubID).Scan(
		&sub.ID, &sub.UserID, &sub.PaddleSubscriptionID, &sub.PaddleCustomerID,
		&sub.Status, &sub.Plan, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSubscriptionNotFound
		}
		return nil, err
	}

	return &sub, nil
}

// Upsert inserts or updates a subscription by user_id.
func (r *SubscriptionRepository) Upsert(ctx context.Context, sub *model.Subscription) error {
	query := `
		INSERT INTO subscriptions (user_id, paddle_subscription_id, paddle_customer_id,
		                           status, plan, current_period_start, current_period_end,
		                           cancel_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			paddle_subscription_id = EXCLUDED.paddle_subscription_id,
			paddle_customer_id = EXCLUDED.paddle_customer_id,
			status = EXCLUDED.status,
			plan = EXCLUDED.plan,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end = EXCLUDED.current_period_end,
			cancel_at = EXCLUDED.cancel_at,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		sub.UserID, sub.PaddleSubscriptionID, sub.PaddleCustomerID,
		sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAt,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
}

// CountUserJobs counts active (non-archived) jobs for a user.
// Only active jobs count against the limit — archived jobs are excluded intentionally.
func (r *SubscriptionRepository) CountUserJobs(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND status = 'active'`, userID,
	).Scan(&count)
	return count, err
}

// CountUserResumes counts all resumes for a user.
// Resumes have no archive concept, so all are counted against the limit.
func (r *SubscriptionRepository) CountUserResumes(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM resumes WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// CountUserApplications counts non-archived applications for a user.
// Archived applications are excluded to match the jobs counting pattern.
func (r *SubscriptionRepository) CountUserApplications(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM applications WHERE user_id = $1 AND status != 'archived'`, userID,
	).Scan(&count)
	return count, err
}

// CountUserAIRequestsThisMonth counts AI match score requests for a user in the current calendar month.
func (r *SubscriptionRepository) CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_usage
		 WHERE user_id = $1
		   AND usage_type = 'match_score'
		   AND created_at >= date_trunc('month', NOW())`, userID,
	).Scan(&count)
	return count, err
}

// CountUserJobParsesThisMonth counts job parse requests for a user in the current calendar month.
func (r *SubscriptionRepository) CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_usage
		 WHERE user_id = $1
		   AND usage_type = 'job_parse'
		   AND created_at >= date_trunc('month', NOW())`, userID,
	).Scan(&count)
	return count, err
}

// RecordAIUsage inserts an AI usage record for a user (match_score type).
func (r *SubscriptionRepository) RecordAIUsage(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (user_id, usage_type) VALUES ($1, 'match_score')`, userID,
	)
	return err
}

// RecordJobParseUsage inserts a job parse usage record for a user.
func (r *SubscriptionRepository) RecordJobParseUsage(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (user_id, usage_type) VALUES ($1, 'job_parse')`, userID,
	)
	return err
}

// GetAllCounts returns all resource counts in a single query (5 sub-selects, 1 round-trip).
func (r *SubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (jobs, resumes, apps, aiReqs, jobParses int, err error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND status = 'active'),
			(SELECT COUNT(*) FROM resumes WHERE user_id = $1),
			(SELECT COUNT(*) FROM applications WHERE user_id = $1 AND status != 'archived'),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type = 'match_score' AND created_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type = 'job_parse' AND created_at >= date_trunc('month', NOW()))
	`
	err = r.pool.QueryRow(ctx, query, userID).Scan(&jobs, &resumes, &apps, &aiReqs, &jobParses)
	return
}
