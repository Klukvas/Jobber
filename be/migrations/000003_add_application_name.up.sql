-- Add name field to applications table
ALTER TABLE applications ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '';

-- Remove default after adding the column (for future inserts to require explicit name)
ALTER TABLE applications ALTER COLUMN name DROP DEFAULT;

-- Add index for name column (for potential search/filtering)
CREATE INDEX idx_applications_name ON applications(name);
