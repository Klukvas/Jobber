-- Drop index
DROP INDEX IF EXISTS idx_resumes_storage_key;

-- Remove storage_key column
ALTER TABLE resumes DROP COLUMN IF EXISTS storage_key;

-- Remove storage_type column
ALTER TABLE resumes DROP COLUMN IF EXISTS storage_type;

-- Drop storage_type enum
DROP TYPE IF EXISTS storage_type;

-- Make file_url NOT NULL again (note: this may fail if there are S3 resumes)
-- Skip this in down migration to prevent data loss
-- ALTER TABLE resumes ALTER COLUMN file_url SET NOT NULL;
