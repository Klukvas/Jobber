-- Revert application status constraint to original values
-- Map new statuses back to old ones
UPDATE applications SET status = 'closed' 
WHERE status IN ('on_hold', 'rejected', 'offer', 'archived');

-- Drop new constraint
ALTER TABLE applications DROP CONSTRAINT applications_status_check;

-- Restore original constraint
ALTER TABLE applications ADD CONSTRAINT applications_status_check 
    CHECK (status IN ('active', 'closed'));
