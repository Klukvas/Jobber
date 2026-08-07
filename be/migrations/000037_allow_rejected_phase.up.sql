-- Rejected becomes a full pipeline phase that can carry stages (e.g.
-- "Requested feedback", "Re-apply later") — the five phases mirror the whole
-- application lifecycle: wishlist → applied → in_progress → offer → rejected.
ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_phase;
ALTER TABLE stage_templates ADD CONSTRAINT chk_stage_templates_phase
    CHECK (phase IN ('wishlist', 'applied', 'in_progress', 'offer', 'rejected'));
