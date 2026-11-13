CREATE TABLE resume_builder_snapshots (
    id UUID PRIMARY KEY,
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    source VARCHAR(20) NOT NULL,
    content JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX resume_builder_snapshots_resume_created_idx
    ON resume_builder_snapshots (resume_builder_id, created_at DESC);
