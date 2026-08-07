-- Move any rejected-phase templates back to in_progress before narrowing the
-- constraint, so the down migration cannot fail on existing data.
UPDATE stage_templates SET phase = 'in_progress' WHERE phase = 'rejected';
ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_phase;
ALTER TABLE stage_templates ADD CONSTRAINT chk_stage_templates_phase
    CHECK (phase IN ('wishlist', 'applied', 'in_progress', 'offer'));
