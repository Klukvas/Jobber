-- Remove name field from applications table
DROP INDEX IF EXISTS idx_applications_name;
ALTER TABLE applications DROP COLUMN IF EXISTS name;
