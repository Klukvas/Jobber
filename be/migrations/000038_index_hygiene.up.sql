-- Index hygiene: drop exact-duplicate indexes and add missing hot-path indexes.
-- All statements are idempotent so the migration is safe to (re)apply.

-- 1) Drop indexes that exactly duplicate a UNIQUE constraint's implicit index.
--    users.email and refresh_tokens.token_hash are already UNIQUE, so these
--    plain indexes only add write overhead.
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;

-- 2) Index the cover_letters foreign keys. Both are ON DELETE SET NULL, so a
--    parent (resume builder / job) delete currently sequential-scans this table.
CREATE INDEX IF NOT EXISTS idx_cover_letters_resume_builder_id
    ON cover_letters (resume_builder_id) WHERE resume_builder_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cover_letters_job_id
    ON cover_letters (job_id) WHERE job_id IS NOT NULL;

-- 3) Composite index for the analytics phase drill-down, which filters on
--    (user_id, phase) — previously only user_id was indexed.
CREATE INDEX IF NOT EXISTS idx_stage_templates_user_phase
    ON stage_templates (user_id, phase);

-- 4) Partial index for due-reminder lookups (is_done = false AND remind_at <= now).
CREATE INDEX IF NOT EXISTS idx_reminders_due
    ON reminders (remind_at) WHERE is_done = false;
