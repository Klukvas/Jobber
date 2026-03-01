CREATE TABLE IF NOT EXISTS match_score_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    resume_id UUID NOT NULL REFERENCES resumes(id) ON DELETE CASCADE,
    result JSONB NOT NULL,
    cached_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_match_score_cache_unique ON match_score_cache (user_id, job_id, resume_id);
CREATE INDEX idx_match_score_cache_job ON match_score_cache (job_id);
CREATE INDEX idx_match_score_cache_resume ON match_score_cache (resume_id);
