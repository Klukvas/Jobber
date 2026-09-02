DROP TABLE IF EXISTS resume_autofill_profiles;

-- Restore the 000039 constraint. Rows with the new usage_type would violate it,
-- so they are removed first — acceptable loss on rollback: those rows only
-- inflate the user's monthly AI count.
DELETE FROM ai_usage WHERE usage_type = 'resume_autofill_parse';
ALTER TABLE ai_usage DROP CONSTRAINT IF EXISTS chk_ai_usage_type;
ALTER TABLE ai_usage ADD CONSTRAINT chk_ai_usage_type
    CHECK (usage_type IN ('match_score', 'job_parse'));
