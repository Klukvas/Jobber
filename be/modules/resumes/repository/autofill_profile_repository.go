package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/jackc/pgx/v5"
)

type AutofillProfileRepository struct {
	pool PgxDB
}

func NewAutofillProfileRepository(pool PgxDB) *AutofillProfileRepository {
	return &AutofillProfileRepository{pool: pool}
}

// Get returns the cached profile, or (nil, nil) when none exists yet.
// parser_version is deliberately not part of the lookup — it is provenance
// metadata only (docs/adr/0001-autofill-profile-economics.md).
func (r *AutofillProfileRepository) Get(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT profile FROM resume_autofill_profiles WHERE resume_id = $1 AND user_id = $2`,
		resumeID, userID,
	).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var profile ai.ParsedResume
	if err := json.Unmarshal(raw, &profile); err != nil {
		// Reads never fall through to extraction (ADR-0001), so a corrupt row
		// would 500 every request forever. Drop it and report a miss — the
		// next request re-extracts. Near-impossible in practice (we wrote the
		// JSONB ourselves); self-healing beats a permanent failure.
		log.Printf("[ERROR] dropping undecodable autofill profile for resume=%s: %v", resumeID, err)
		if _, delErr := r.pool.Exec(ctx,
			`DELETE FROM resume_autofill_profiles WHERE resume_id = $1 AND user_id = $2`,
			resumeID, userID,
		); delErr != nil {
			return nil, fmt.Errorf("failed to drop undecodable autofill profile: %w", delErr)
		}
		return nil, nil
	}
	return &profile, nil
}

// Upsert stores the profile idempotently — concurrent first extractions of the
// same resume both succeed, last write wins. Returns whether a NEW row was
// created: `xmax = 0` is true only for a freshly inserted tuple, so the caller
// can charge quota exactly once per file even when two extractions race.
func (r *AutofillProfileRepository) Upsert(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
	raw, err := json.Marshal(profile)
	if err != nil {
		return false, fmt.Errorf("failed to encode autofill profile: %w", err)
	}

	var inserted bool
	err = r.pool.QueryRow(ctx,
		`INSERT INTO resume_autofill_profiles (resume_id, user_id, profile, parser_version)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (resume_id) DO UPDATE
		 SET profile = EXCLUDED.profile, parser_version = EXCLUDED.parser_version, updated_at = NOW()
		 RETURNING (xmax = 0)`,
		resumeID, userID, raw, parserVersion,
	).Scan(&inserted)
	if err != nil {
		return false, err
	}
	return inserted, nil
}

func (r *AutofillProfileRepository) InvalidateByResume(ctx context.Context, resumeID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM resume_autofill_profiles WHERE resume_id = $1`, resumeID,
	)
	return err
}
