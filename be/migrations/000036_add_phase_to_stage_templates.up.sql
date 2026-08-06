-- Fixed pipeline phases that contain user-defined stages. Rejected is
-- terminal and never carries stages, so it is not a valid template phase.
ALTER TABLE stage_templates ADD COLUMN phase TEXT NOT NULL DEFAULT 'in_progress';

ALTER TABLE stage_templates ADD CONSTRAINT chk_stage_templates_phase
    CHECK (phase IN ('wishlist', 'applied', 'in_progress', 'offer'));

-- Backfill existing templates:
--  * order=1 templates mirror the synthetic "Applied" funnel bucket → applied
--  * templates named like "offer" are outcome steps → offer
--  * everything else is an interview step → in_progress (column default)
UPDATE stage_templates SET phase = 'applied' WHERE "order" = 1;
UPDATE stage_templates SET phase = 'offer' WHERE "order" > 1 AND name ILIKE 'offer%';
