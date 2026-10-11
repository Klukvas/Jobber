-- Semantic stage kinds.
--
-- The user owns a column's name and its position; analytics needs to know what
-- a column *means*. Deriving that from the label ("%interview%") or the order
-- ("anything past the first column is a response") breaks the moment someone
-- renames or reorders their board — a renamed "Interview" stopped being an
-- interview, and a custom "Follow-up reminder" column counted as a reply from
-- the employer. `kind` makes the meaning explicit and stable.
--
-- 'other' is the default: a column whose meaning we cannot infer carries no
-- analytics weight until the user classifies it in the stage UI. That is the
-- conservative direction — it under-reports rather than inventing responses.
BEGIN;

-- Mirrors 000040: a data backfill is not a user edit, so updated_at must not
-- move. The trigger is re-enabled before COMMIT.
ALTER TABLE stage_templates DISABLE TRIGGER update_stage_templates_updated_at;

ALTER TABLE stage_templates ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'other';

ALTER TABLE stage_templates DROP CONSTRAINT IF EXISTS chk_stage_templates_kind;
ALTER TABLE stage_templates ADD CONSTRAINT chk_stage_templates_kind
    CHECK (kind IN ('saved', 'applied', 'response', 'interview', 'offer', 'rejected', 'other'));

-- Backfill: only the names this app has ever seeded itself (migrations 000001+
-- and be/cmd/seed) are recognized, matched case-insensitively and trimmed.
-- Anything else — including a user's own column that merely sounds similar —
-- stays 'other' on purpose.
--
-- The seeded pipelines are:
--   Wishlist · Applied · Screening · Technical Interview · Final Interview ·
--   Offer · Rejected        (default pipeline)
--   Wishlist · Applied · Interview · Offer · Rejected   (reviewer pipeline)
UPDATE stage_templates SET kind = 'saved'     WHERE LOWER(BTRIM(name)) = 'wishlist';
UPDATE stage_templates SET kind = 'applied'   WHERE LOWER(BTRIM(name)) = 'applied';
UPDATE stage_templates SET kind = 'response'  WHERE LOWER(BTRIM(name)) = 'screening';
UPDATE stage_templates SET kind = 'interview' WHERE LOWER(BTRIM(name)) IN
    ('interview', 'technical interview', 'final interview');
UPDATE stage_templates SET kind = 'offer'     WHERE LOWER(BTRIM(name)) = 'offer';
UPDATE stage_templates SET kind = 'rejected'  WHERE LOWER(BTRIM(name)) = 'rejected';

-- Analytics filters columns by (user_id, kind) on every response/interview
-- metric, the same access pattern the dropped (user_id, phase) index served.
CREATE INDEX IF NOT EXISTS idx_stage_templates_user_kind
    ON stage_templates (user_id, kind);

ALTER TABLE stage_templates ENABLE TRIGGER update_stage_templates_updated_at;

COMMIT;
