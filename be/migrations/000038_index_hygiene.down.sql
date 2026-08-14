-- Reverse 000038: drop the added indexes and recreate the duplicates.
DROP INDEX IF EXISTS idx_reminders_due;
DROP INDEX IF EXISTS idx_stage_templates_user_phase;
DROP INDEX IF EXISTS idx_cover_letters_job_id;
DROP INDEX IF EXISTS idx_cover_letters_resume_builder_id;

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
