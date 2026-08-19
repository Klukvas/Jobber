-- Repair a LOCAL DEV database stuck on migration 000041 (single-pipeline finalize).
--
-- Symptom: the API refuses to boot with
--   "Failed to run database migrations ... column current_stage_template_id of
--    relation jobs contains null values (23502)"
--   and `schema_migrations` shows version 41, dirty = true.
--
-- Cause: migration 040 added a NULLABLE current_stage_template_id and, at the
-- time, a startup backfill (internal/backfill) populated it before 041 enforced
-- NOT NULL. That backfill was removed in the cleanup commit, so a dev DB that
-- still had pre-single-pipeline cards (created before 040, never backfilled)
-- reaches 041 with NULL columns and the SET NOT NULL fails. Fresh DBs are fine
-- (no rows to violate the constraint), so this only bites long-lived local DBs.
--
-- This script does exactly what the old backfill + the rest of 041 would do,
-- additively (it never deletes data) and idempotently (safe to re-run):
--   1. give any owner of a stage-less card a default "Applied" column,
--   2. drop every NULL card into its owner's lowest-order column,
--   3. apply the remaining 041 DDL, and
--   4. clear the dirty flag.
--
-- DEV ONLY. Prod is already at 41:0 — never run this against production.
--
-- Run:  make repair-dev-db            (from be/, honours DATABASE_URL like migrate-up;
--                                       remember the local docker port is 5434, e.g.
--                                       make repair-dev-db DATABASE_URL="postgresql://jobber:jobber@localhost:5434/jobber?sslmode=disable")
-- Or:   docker exec -i jobber-postgres psql -U jobber -d jobber -v ON_ERROR_STOP=1 < scripts/repair-dev-db-041.sql

BEGIN;

-- 1. Owners of stage-less cards but no pipeline at all get a default column.
INSERT INTO stage_templates (user_id, name, "order")
SELECT DISTINCT j.user_id, 'Applied', 1
FROM jobs j
WHERE j.current_stage_template_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM stage_templates st WHERE st.user_id = j.user_id);

-- 2. Backfill every NULL card into its owner's lowest-order column.
UPDATE jobs j
SET current_stage_template_id = pick.id
FROM (
  SELECT DISTINCT ON (user_id) user_id, id
  FROM stage_templates
  ORDER BY user_id, "order" ASC
) pick
WHERE j.user_id = pick.user_id
  AND j.current_stage_template_id IS NULL;

-- 3. The invariant holds now — apply the rest of migration 041 (idempotent).
ALTER TABLE jobs ALTER COLUMN current_stage_template_id SET NOT NULL;
ALTER TABLE jobs DROP COLUMN IF EXISTS status;
DROP INDEX IF EXISTS idx_stage_templates_user_phase;
ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_phase;
ALTER TABLE stage_templates DROP COLUMN IF EXISTS phase;

-- 4. 41 is fully applied — clear the dirty flag.
UPDATE schema_migrations SET dirty = false WHERE version = 41;

COMMIT;

-- Sanity check (expect: null_stage_jobs = 0, migration = 41 dirty=false).
SELECT 'null_stage_jobs' AS check, count(*)::text AS value FROM jobs WHERE current_stage_template_id IS NULL
UNION ALL
SELECT 'migration', version::text || ' dirty=' || dirty::text FROM schema_migrations;
