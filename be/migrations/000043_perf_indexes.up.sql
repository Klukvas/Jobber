-- Performance indexes for the hot board + analytics paths.
--
-- 1. idx_jobs_user_archived: migration 041 dropped jobs.status, which cascaded
--    the composite idx_jobs_user_status(user_id, status) that used to cover the
--    board filter. The board now filters `WHERE user_id = $1 AND is_archived = ?`
--    with only idx_jobs_user_id(user_id) available; this composite restores an
--    index-only path for both the default (false) and archive (true) views.
-- 2. idx_job_stages_job_stage: the analytics queries all run
--    `EXISTS (... job_stages WHERE job_id = ? AND stage_template_id = ?)`; a
--    two-column index resolves that with a single lookup.
--
-- NOTE: not CONCURRENTLY because golang-migrate wraps each file in a tx and these
-- tables are small at deploy time. Add CONCURRENTLY (outside a tx block) if either
-- table ever grows past ~1M rows.
BEGIN;

CREATE INDEX IF NOT EXISTS idx_jobs_user_archived ON jobs (user_id, is_archived);

CREATE INDEX IF NOT EXISTS idx_job_stages_job_stage ON job_stages (job_id, stage_template_id);

COMMIT;
