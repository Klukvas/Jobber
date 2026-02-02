-- Add storage type enum
CREATE TYPE storage_type AS ENUM ('external', 's3');

-- Add storage_type and storage_key columns to resumes table
ALTER TABLE resumes
ADD COLUMN storage_type storage_type NOT NULL DEFAULT 'external',
ADD COLUMN storage_key TEXT;

-- Make file_url nullable since S3 resumes won't have external URLs
ALTER TABLE resumes
ALTER COLUMN file_url DROP NOT NULL;

-- Add index for storage_key lookups
CREATE INDEX idx_resumes_storage_key ON resumes(storage_key) WHERE storage_key IS NOT NULL;

-- Update existing resumes to have storage_type 'external'
UPDATE resumes SET storage_type = 'external' WHERE file_url IS NOT NULL;
