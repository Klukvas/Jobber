-- Constrain ai_usage.usage_type to the known values so a typo in Go code can't
-- silently insert an uncounted row (which would bypass usage limits).
-- Existing data verified to contain only 'match_score' and 'job_parse'.
ALTER TABLE ai_usage DROP CONSTRAINT IF EXISTS chk_ai_usage_type;
ALTER TABLE ai_usage ADD CONSTRAINT chk_ai_usage_type
    CHECK (usage_type IN ('match_score', 'job_parse'));
