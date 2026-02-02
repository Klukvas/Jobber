-- Remove index on status
DROP INDEX IF EXISTS idx_jobs_status;

-- Remove status column from jobs table
ALTER TABLE jobs
DROP COLUMN IF EXISTS status;
