ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_phase;
ALTER TABLE stage_templates DROP COLUMN IF EXISTS phase;
