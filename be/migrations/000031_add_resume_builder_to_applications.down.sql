ALTER TABLE applications DROP CONSTRAINT IF EXISTS chk_single_resume;
DROP INDEX IF EXISTS idx_applications_resume_builder_id;
ALTER TABLE applications DROP COLUMN IF EXISTS resume_builder_id;
