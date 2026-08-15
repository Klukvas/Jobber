-- Single-pipeline refactor, DEPLOY 1 (transitional).
--
-- Adds the single-axis pointer (current_stage_template_id) and the orthogonal
-- is_archived flag. Deliberately KEEPS jobs.status (+ its CHECK) and
-- stage_templates.phase so a rolled-back OLD image still runs against this
-- schema; both are dropped later in 000041, after the new code+FE are live.
--
-- The per-card column pointer is populated by the Go backfill (be/cmd/backfill)
-- run right after this migration; here we only add the (nullable) column.
--
-- Explicit BEGIN/COMMIT + trigger disable, mirroring 000034: single-transaction
-- under any runner, and the is_archived bulk UPDATE must not bump updated_at.
BEGIN;

ALTER TABLE jobs DISABLE TRIGGER update_jobs_updated_at;

-- Which pipeline column (stage_template) the card currently sits in — the only
-- axis in the new model. ON DELETE RESTRICT: a column with cards can't be
-- deleted (Huntr behavior). Nullable now; the backfill fills it and 000041
-- makes it NOT NULL.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS current_stage_template_id UUID
    REFERENCES stage_templates(id) ON DELETE RESTRICT;

-- Orthogonal archived flag (hides a card from board + funnel), replacing the
-- old status='archived'.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT false;

-- Raise the flag for currently-archived cards. status column stays untouched.
UPDATE jobs SET is_archived = true WHERE status = 'archived';

CREATE INDEX IF NOT EXISTS idx_jobs_current_stage_template_id
    ON jobs(current_stage_template_id) WHERE current_stage_template_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_is_archived
    ON jobs(user_id) WHERE is_archived = true;

ALTER TABLE jobs ENABLE TRIGGER update_jobs_updated_at;

COMMIT;
