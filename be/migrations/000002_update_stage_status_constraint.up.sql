-- Drop the old check constraint
ALTER TABLE application_stages
DROP CONSTRAINT application_stages_status_check;

-- Add the new check constraint with all supported statuses
ALTER TABLE application_stages
ADD CONSTRAINT application_stages_status_check
CHECK (status IN ('pending', 'active', 'completed', 'skipped', 'cancelled'));
