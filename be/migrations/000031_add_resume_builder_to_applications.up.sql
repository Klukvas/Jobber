ALTER TABLE applications ADD COLUMN resume_builder_id UUID REFERENCES resume_builders(id) ON DELETE SET NULL;
CREATE INDEX idx_applications_resume_builder_id ON applications(resume_builder_id) WHERE resume_builder_id IS NOT NULL;
ALTER TABLE applications ADD CONSTRAINT chk_single_resume CHECK (resume_id IS NULL OR resume_builder_id IS NULL);
