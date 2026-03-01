DROP INDEX IF EXISTS idx_companies_user_favorite;
DROP INDEX IF EXISTS idx_jobs_user_favorite;
ALTER TABLE companies DROP COLUMN IF EXISTS is_favorite;
ALTER TABLE jobs DROP COLUMN IF EXISTS is_favorite;
