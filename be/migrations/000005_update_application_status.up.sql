-- Update application status constraint to support new status values
-- Old: 'active', 'closed'
-- New: 'active', 'on_hold', 'rejected', 'offer', 'archived'

-- Drop existing constraint
ALTER TABLE applications DROP CONSTRAINT applications_status_check;

-- Add new constraint with expanded status values
ALTER TABLE applications ADD CONSTRAINT applications_status_check 
    CHECK (status IN ('active', 'on_hold', 'rejected', 'offer', 'archived'));

-- Migrate existing 'closed' status to 'archived'
UPDATE applications SET status = 'archived' WHERE status = 'closed';
