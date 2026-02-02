-- Add status column to jobs table
ALTER TABLE jobs
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived'));

-- Create index on status for faster filtering
CREATE INDEX idx_jobs_status ON jobs(status);
