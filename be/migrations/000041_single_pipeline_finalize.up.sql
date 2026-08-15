-- Single-pipeline refactor, DEPLOY 2 (finalize). Run only after deploy 1 (the
-- new phase-free / status-free code + the 040 backfill) is verified live, so no
-- old image can be rolled back to.
--
-- The backfill guaranteed every job has a column, so we can enforce NOT NULL and
-- drop the now-dead status/phase columns.
BEGIN;

-- Every card is in a column now.
ALTER TABLE jobs ALTER COLUMN current_stage_template_id SET NOT NULL;

-- Drop the dead job status (the column IS the state). DROP COLUMN cascades its
-- CHECK constraint and any single-column indexes.
ALTER TABLE jobs DROP COLUMN IF EXISTS status;

-- Drop the dead stage phase (+ its index from 038 and CHECK from 036/037).
DROP INDEX IF EXISTS idx_stage_templates_user_phase;
ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_phase;
ALTER TABLE stage_templates DROP COLUMN IF EXISTS phase;

COMMIT;
