-- Index for JOIN applications → resumes (used in enriched application list)
CREATE INDEX IF NOT EXISTS idx_applications_resume_id ON applications(resume_id);

-- Index for JOIN application_stages → stage_templates (used in stage lookups)
CREATE INDEX IF NOT EXISTS idx_application_stages_stage_template_id ON application_stages(stage_template_id);

-- Composite index for ai_usage monthly count queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_user_type_month ON ai_usage(user_id, usage_type, created_at);
