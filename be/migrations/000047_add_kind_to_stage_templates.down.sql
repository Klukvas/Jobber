-- Reverses 000047: the classification is derived data (names and order are
-- untouched by the up migration), so dropping the column loses only the
-- user's own reclassifications.
BEGIN;

DROP INDEX IF EXISTS idx_stage_templates_user_kind;
ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_kind;
ALTER TABLE stage_templates DROP COLUMN IF EXISTS kind;

COMMIT;
