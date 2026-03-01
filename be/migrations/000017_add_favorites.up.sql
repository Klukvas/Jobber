ALTER TABLE jobs ADD COLUMN is_favorite BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE companies ADD COLUMN is_favorite BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX idx_jobs_user_favorite ON jobs(user_id, is_favorite) WHERE is_favorite = true;
CREATE INDEX idx_companies_user_favorite ON companies(user_id, is_favorite) WHERE is_favorite = true;
