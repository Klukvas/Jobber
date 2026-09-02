-- Autofill Profiles extracted from Uploaded Resumes (docs/plans/autofill-uploaded-pdf.md).
-- One row per resume file; profile is the raw AI extraction (ai.ParsedResume).
-- parser_version is provenance metadata only — reads must NOT filter by it
-- (bumping it must not silently invalidate profiles users already paid quota
-- for, see docs/adr/0001-autofill-profile-economics.md).
CREATE TABLE IF NOT EXISTS resume_autofill_profiles (
    resume_id UUID PRIMARY KEY REFERENCES resumes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile JSONB NOT NULL,
    parser_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Not used by application queries today (Get/Upsert hit the resume_id PK) —
-- exists for the users-delete FK cascade and future per-user listings.
CREATE INDEX IF NOT EXISTS idx_resume_autofill_profiles_user_id ON resume_autofill_profiles(user_id);

-- Widen the usage_type guard from 000039 for the new extraction usage type.
ALTER TABLE ai_usage DROP CONSTRAINT IF EXISTS chk_ai_usage_type;
ALTER TABLE ai_usage ADD CONSTRAINT chk_ai_usage_type
    CHECK (usage_type IN ('match_score', 'job_parse', 'resume_autofill_parse'));
