BEGIN;

DROP INDEX IF EXISTS idx_job_stages_job_stage;
DROP INDEX IF EXISTS idx_jobs_user_archived;

COMMIT;
