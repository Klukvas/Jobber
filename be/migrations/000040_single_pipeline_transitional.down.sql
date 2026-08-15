-- Reverse 000040. Best-effort (dev/test); prod recovery is the pre-deploy dump.
BEGIN;

ALTER TABLE jobs DISABLE TRIGGER update_jobs_updated_at;

DROP INDEX IF EXISTS idx_jobs_is_archived;
DROP INDEX IF EXISTS idx_jobs_current_stage_template_id;

-- Collapse the flag back into status='archived'.
UPDATE jobs SET status = 'archived' WHERE is_archived = true;

ALTER TABLE jobs DROP COLUMN IF EXISTS is_archived;
ALTER TABLE jobs DROP COLUMN IF EXISTS current_stage_template_id;

ALTER TABLE jobs ENABLE TRIGGER update_jobs_updated_at;

COMMIT;
