-- Revert to the original check constraint (only pending and completed)
ALTER TABLE application_stages
DROP CONSTRAINT application_stages_status_check;

ALTER TABLE application_stages
ADD CONSTRAINT application_stages_status_check
CHECK (status IN ('pending', 'completed'));
