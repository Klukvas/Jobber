-- Reverse 000041. Best-effort (dev/test); the original status/phase values are
-- unrecoverable — prod recovery is the pre-deploy dump.
BEGIN;

ALTER TABLE jobs ALTER COLUMN current_stage_template_id DROP NOT NULL;

-- Re-add phase (000037 definition) + its 000038 index.
ALTER TABLE stage_templates ADD COLUMN IF NOT EXISTS phase TEXT NOT NULL DEFAULT 'in_progress';
ALTER TABLE stage_templates ADD CONSTRAINT chk_stage_templates_phase
    CHECK (phase IN ('wishlist', 'applied', 'in_progress', 'offer', 'rejected'));
CREATE INDEX IF NOT EXISTS idx_stage_templates_user_phase
    ON stage_templates (user_id, phase);

-- Re-add status with the old enum; collapse the archived flag back into it.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'saved';
UPDATE jobs SET status = 'archived' WHERE is_archived = true;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('saved', 'applied', 'on_hold', 'offer', 'rejected', 'archived'));

COMMIT;
