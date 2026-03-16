-- Add text_color column as nullable first, backfill, then enforce NOT NULL + default.
-- This avoids a window where new rows could get the hardcoded default instead of primary_color.
ALTER TABLE resume_builders ADD COLUMN text_color VARCHAR(7);

UPDATE resume_builders SET text_color = primary_color;

ALTER TABLE resume_builders ALTER COLUMN text_color SET NOT NULL;
ALTER TABLE resume_builders ALTER COLUMN text_color SET DEFAULT '#2563eb';
