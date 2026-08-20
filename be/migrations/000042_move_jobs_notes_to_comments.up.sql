-- Move every job's free-form notes into the comments module before dropping the
-- column. Comments (per-job and per-stage) are now the single place for
-- free-form annotations on an application. Company notes are unrelated and stay.
BEGIN;

-- Backfill: one stage-less (application-level) comment per job that has a
-- non-empty note. created_at is stamped from the job so migrated notes don't
-- push every job's last-activity to "now" (the board derives last_activity from
-- the newest comment time). The BEFORE UPDATE trigger on comments does not fire
-- on INSERT, so the explicit timestamps are preserved.
INSERT INTO comments (user_id, job_id, stage_id, content, created_at, updated_at)
SELECT j.user_id, j.id, NULL, btrim(j.notes), j.created_at, j.created_at
FROM jobs j
WHERE j.notes IS NOT NULL AND btrim(j.notes) <> '';

ALTER TABLE jobs DROP COLUMN IF EXISTS notes;

COMMIT;
