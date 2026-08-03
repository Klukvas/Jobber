-- BEST-EFFORT rollback for dev/test cycles only.
-- The merge is lossy (original application UUIDs are gone, application.name is
-- collapsed into jobs.title, a job's pre-application wishlist state is collapsed
-- into one row). PRODUCTION rollback = restore the pre-deploy DB backup.
-- This script restores the pre-merge SCHEMA and an approximation of the data:
--   * every job with applied_at IS NOT NULL becomes one application (new UUID)
--   * stages/comments/reminders on jobs WITHOUT applied_at are DELETED
--     (they cannot exist in the old schema, which requires an application)

BEGIN;

ALTER TABLE jobs      DISABLE TRIGGER update_jobs_updated_at;
ALTER TABLE comments  DISABLE TRIGGER update_comments_updated_at;
ALTER TABLE reminders DISABLE TRIGGER update_reminders_updated_at;

-- (1) recreate applications (shape of 000001+000003+000005+000019+000031)
CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    resume_id UUID REFERENCES resumes(id) ON DELETE SET NULL,
    resume_builder_id UUID REFERENCES resume_builders(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    current_stage_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'on_hold', 'rejected', 'offer', 'archived')),
    applied_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_single_resume CHECK (resume_id IS NULL OR resume_builder_id IS NULL)
);

CREATE TEMP TABLE job_app_map ON COMMIT DROP AS
SELECT id AS job_id, uuid_generate_v4() AS app_id
FROM jobs WHERE applied_at IS NOT NULL;

INSERT INTO applications (id, user_id, job_id, resume_id, resume_builder_id, name,
                          current_stage_id, status, applied_at, created_at, updated_at)
SELECT m.app_id, j.user_id, j.id, j.resume_id, j.resume_builder_id, j.title,
       j.current_stage_id,
       CASE j.status WHEN 'applied' THEN 'active' ELSE j.status END,
       j.applied_at, j.created_at, j.updated_at
FROM jobs j
JOIN job_app_map m ON m.job_id = j.id;

-- (2) rename back and repoint children to the reconstructed applications
ALTER TABLE jobs DROP CONSTRAINT fk_jobs_current_stage;
ALTER TABLE job_stages DROP CONSTRAINT job_stages_job_id_fkey;
ALTER TABLE comments   DROP CONSTRAINT comments_job_id_fkey;
ALTER TABLE reminders  DROP CONSTRAINT reminders_job_id_fkey;

ALTER TABLE job_stages RENAME TO application_stages;
ALTER TABLE application_stages RENAME COLUMN job_id TO application_id;
ALTER TABLE comments  RENAME COLUMN job_id TO application_id;
ALTER TABLE reminders RENAME COLUMN job_id TO application_id;

UPDATE application_stages s SET application_id = m.app_id
FROM job_app_map m WHERE s.application_id = m.job_id;
UPDATE comments c SET application_id = m.app_id
FROM job_app_map m WHERE c.application_id = m.job_id;
UPDATE reminders r SET application_id = m.app_id
FROM job_app_map m WHERE r.application_id = m.job_id;

-- children of never-applied jobs cannot exist in the old schema
DELETE FROM application_stages WHERE application_id NOT IN (SELECT id FROM applications);
DELETE FROM comments  WHERE application_id NOT IN (SELECT id FROM applications);
DELETE FROM reminders WHERE application_id NOT IN (SELECT id FROM applications);

ALTER TABLE application_stages ADD CONSTRAINT application_stages_application_id_fkey
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT comments_application_id_fkey
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;
ALTER TABLE reminders ADD CONSTRAINT reminders_application_id_fkey
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE;
ALTER TABLE applications ADD CONSTRAINT fk_applications_current_stage
    FOREIGN KEY (current_stage_id) REFERENCES application_stages(id) ON DELETE SET NULL;

-- (3) restore jobs to the old shape
ALTER TABLE jobs DROP CONSTRAINT jobs_status_check;
UPDATE jobs SET status = CASE WHEN status = 'archived' THEN 'archived' ELSE 'active' END;
ALTER TABLE jobs ALTER COLUMN status SET DEFAULT 'active';
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check CHECK (status IN ('active', 'archived'));

ALTER TABLE jobs DROP CONSTRAINT chk_jobs_single_resume;
ALTER TABLE jobs DROP COLUMN applied_at;
ALTER TABLE jobs DROP COLUMN resume_id;
ALTER TABLE jobs DROP COLUMN resume_builder_id;
ALTER TABLE jobs DROP COLUMN current_stage_id;

ALTER TABLE jobs ADD COLUMN board_column VARCHAR(20) NOT NULL DEFAULT 'wishlist'
    CHECK (board_column IN ('wishlist', 'applied', 'interview', 'offer', 'rejected'));
CREATE INDEX idx_jobs_board_column ON jobs(user_id, board_column);

-- (4) restore names, indexes, enum, triggers
ALTER INDEX idx_job_stages_job_id            RENAME TO idx_application_stages_application_id;
ALTER INDEX idx_job_stages_status            RENAME TO idx_application_stages_status;
ALTER INDEX idx_job_stages_order             RENAME TO idx_application_stages_order;
ALTER INDEX idx_job_stages_stage_template_id RENAME TO idx_application_stages_stage_template_id;
ALTER INDEX idx_comments_job_id              RENAME TO idx_comments_application_id;
ALTER INDEX idx_reminders_job_id             RENAME TO idx_reminders_application_id;
ALTER TABLE application_stages RENAME CONSTRAINT job_stages_status_check TO application_stages_status_check;
ALTER TABLE application_stages RENAME CONSTRAINT job_stages_pkey TO application_stages_pkey;

CREATE INDEX idx_applications_user_id ON applications(user_id);
CREATE INDEX idx_applications_job_id ON applications(job_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_applied_at ON applications(applied_at);
CREATE INDEX idx_applications_name ON applications(name);
CREATE INDEX idx_applications_resume_id ON applications(resume_id);
CREATE INDEX idx_applications_resume_builder_id ON applications(resume_builder_id)
    WHERE resume_builder_id IS NOT NULL;

ALTER TABLE tag_relations DROP CONSTRAINT tag_relations_entity_type_check;
ALTER TABLE tag_relations ADD CONSTRAINT tag_relations_entity_type_check
    CHECK (entity_type IN ('application', 'job', 'company'));

CREATE TRIGGER update_applications_updated_at BEFORE UPDATE ON applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_cleanup_tag_relations_applications
    BEFORE DELETE ON applications
    FOR EACH ROW EXECUTE FUNCTION cleanup_tag_relations('application');

-- idx_jobs_applied_at/resume_id/resume_builder_id/current_stage_id/user_applied
-- were dropped automatically with their columns in step (3);
-- idx_jobs_user_status references surviving columns, so drop it explicitly.
DROP INDEX IF EXISTS idx_jobs_user_status;

ALTER TABLE jobs      ENABLE TRIGGER update_jobs_updated_at;
ALTER TABLE comments  ENABLE TRIGGER update_comments_updated_at;
ALTER TABLE reminders ENABLE TRIGGER update_reminders_updated_at;

COMMIT;
