-- Merge applications into jobs: one entity with a pipeline status.
-- Explicit BEGIN/COMMIT: guarantees single-transaction execution (and a live
-- ON COMMIT DROP temp table) under any runner — golang-migrate's single Exec
-- as well as plain psql -f. Any failure rolls back everything.
--
-- Mapping rules:
--   * job with 0 applications: status 'active' -> 'saved', 'archived' stays; applied_at NULL
--   * newest application (per job) merges INTO the original jobs row, preserving job.id
--     so match_score_cache, cover_letters.job_id and tag_relations('job') stay valid
--   * each older application becomes a cloned jobs row whose id = the application id,
--     so its stages/comments/reminders need no repointing
--   * application.status 'active' -> 'applied'; other statuses carry over
--   * "is an application" predicate after the merge: applied_at IS NOT NULL

BEGIN;

-- (0) updated_at triggers would clobber timestamps on migration UPDATEs
ALTER TABLE jobs      DISABLE TRIGGER update_jobs_updated_at;
ALTER TABLE comments  DISABLE TRIGGER update_comments_updated_at;
ALTER TABLE reminders DISABLE TRIGGER update_reminders_updated_at;

-- (1) reshape jobs
ALTER TABLE jobs DROP COLUMN board_column;
ALTER TABLE jobs ADD COLUMN applied_at        TIMESTAMP;
ALTER TABLE jobs ADD COLUMN resume_id         UUID;
ALTER TABLE jobs ADD COLUMN resume_builder_id UUID;
ALTER TABLE jobs ADD COLUMN current_stage_id  UUID;
ALTER TABLE jobs DROP CONSTRAINT jobs_status_check;

-- (2) application -> target job mapping (rn=1 keeps the original job row)
CREATE TEMP TABLE app_map ON COMMIT DROP AS
SELECT id AS app_id,
       job_id,
       CASE WHEN rn = 1 THEN job_id ELSE id END AS target_job_id,
       rn
FROM (
    SELECT id, job_id,
           ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY created_at DESC, id DESC) AS rn
    FROM applications
) ranked;

-- (3) one cloned job row per older application (id := application id).
--     MUST run before step (4): clones copy the PRISTINE job fields, and the
--     merge update below overwrites title/status on the original row.
INSERT INTO jobs (id, user_id, company_id, title, source, url, notes, description,
                  is_favorite, status, applied_at, resume_id, resume_builder_id,
                  current_stage_id, created_at, updated_at)
SELECT a.id, a.user_id, j.company_id,
       COALESCE(NULLIF(a.name, ''), j.title),
       j.source, j.url, j.notes, j.description, false,
       CASE a.status WHEN 'active' THEN 'applied' ELSE a.status END,
       a.applied_at, a.resume_id, a.resume_builder_id, a.current_stage_id,
       a.created_at, a.updated_at
FROM applications a
JOIN app_map m ON m.app_id = a.id AND m.rn > 1
JOIN jobs j ON j.id = a.job_id;

-- (4) newest application merges into the original job row
UPDATE jobs j SET
    title             = COALESCE(NULLIF(a.name, ''), j.title),
    status            = CASE a.status WHEN 'active' THEN 'applied' ELSE a.status END,
    applied_at        = a.applied_at,
    resume_id         = a.resume_id,
    resume_builder_id = a.resume_builder_id,
    current_stage_id  = a.current_stage_id,
    updated_at        = GREATEST(j.updated_at, a.updated_at)
FROM applications a
JOIN app_map m ON m.app_id = a.id AND m.rn = 1
WHERE j.id = a.job_id;

-- (5) jobs that had no applications become wishlist cards
UPDATE jobs SET status = 'saved' WHERE status = 'active';

ALTER TABLE jobs ALTER COLUMN status SET DEFAULT 'saved';
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('saved', 'applied', 'on_hold', 'offer', 'rejected', 'archived'));

-- (6) repoint children of merged (rn=1) applications; drop FKs first
ALTER TABLE application_stages DROP CONSTRAINT application_stages_application_id_fkey;
ALTER TABLE comments           DROP CONSTRAINT comments_application_id_fkey;
ALTER TABLE reminders          DROP CONSTRAINT reminders_application_id_fkey;

UPDATE application_stages s SET application_id = m.target_job_id
FROM app_map m WHERE s.application_id = m.app_id AND m.rn = 1;

UPDATE comments c SET application_id = m.target_job_id
FROM app_map m WHERE c.application_id = m.app_id AND m.rn = 1;

UPDATE reminders r SET application_id = m.target_job_id
FROM app_map m WHERE r.application_id = m.app_id AND m.rn = 1;

-- (7) tag_relations: move 'application' tags to the target job, then narrow the enum
INSERT INTO tag_relations (id, tag_id, entity_type, entity_id, created_at)
SELECT uuid_generate_v4(), tr.tag_id, 'job', m.target_job_id, tr.created_at
FROM tag_relations tr
JOIN app_map m ON m.app_id = tr.entity_id
WHERE tr.entity_type = 'application'
ON CONFLICT (tag_id, entity_type, entity_id) DO NOTHING;

DELETE FROM tag_relations WHERE entity_type = 'application';

ALTER TABLE tag_relations DROP CONSTRAINT tag_relations_entity_type_check;
ALTER TABLE tag_relations ADD CONSTRAINT tag_relations_entity_type_check
    CHECK (entity_type IN ('job', 'company'));

-- (8) sanity: every application's target job must exist AND be marked applied.
-- Set-membership (not just aggregate counts) catches a positional error where a
-- missing merge and an extra clone would otherwise cancel out in a plain count.
DO $$
DECLARE
    missing BIGINT;
BEGIN
    SELECT COUNT(*) INTO missing
    FROM app_map m
    WHERE NOT EXISTS (
        SELECT 1 FROM jobs j
        WHERE j.id = m.target_job_id AND j.applied_at IS NOT NULL
    );
    IF missing > 0 THEN
        RAISE EXCEPTION 'merge mismatch: % application(s) have no corresponding applied job', missing;
    END IF;
END $$;

-- (9) drop applications (takes update_applications_updated_at,
--     trg_cleanup_tag_relations_applications, fk_applications_current_stage and
--     idx_applications_* with it), then rename tables/columns
DROP TABLE applications;

ALTER TABLE application_stages RENAME TO job_stages;
ALTER TABLE job_stages RENAME COLUMN application_id TO job_id;
ALTER TABLE comments   RENAME COLUMN application_id TO job_id;
ALTER TABLE reminders  RENAME COLUMN application_id TO job_id;

-- (10) new FKs (creation doubles as validation of steps 3-6)
ALTER TABLE job_stages ADD CONSTRAINT job_stages_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT comments_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;
ALTER TABLE reminders ADD CONSTRAINT reminders_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;
ALTER TABLE jobs ADD CONSTRAINT fk_jobs_current_stage
    FOREIGN KEY (current_stage_id) REFERENCES job_stages(id) ON DELETE SET NULL;
-- SET NULL (not RESTRICT): applications may exist without a resume since 000019
ALTER TABLE jobs ADD CONSTRAINT jobs_resume_id_fkey
    FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE SET NULL;
ALTER TABLE jobs ADD CONSTRAINT jobs_resume_builder_id_fkey
    FOREIGN KEY (resume_builder_id) REFERENCES resume_builders(id) ON DELETE SET NULL;
-- at most one resume kind per job (was chk_single_resume on applications, 000031)
ALTER TABLE jobs ADD CONSTRAINT chk_jobs_single_resume
    CHECK (resume_id IS NULL OR resume_builder_id IS NULL);

-- (11) rename surviving indexes/constraints; recreate the ones lost with applications
ALTER INDEX idx_application_stages_application_id    RENAME TO idx_job_stages_job_id;
ALTER INDEX idx_application_stages_status            RENAME TO idx_job_stages_status;
ALTER INDEX idx_application_stages_order             RENAME TO idx_job_stages_order;
ALTER INDEX idx_application_stages_stage_template_id RENAME TO idx_job_stages_stage_template_id;
ALTER INDEX idx_comments_application_id              RENAME TO idx_comments_job_id;
ALTER INDEX idx_reminders_application_id             RENAME TO idx_reminders_job_id;
ALTER TABLE job_stages RENAME CONSTRAINT application_stages_status_check TO job_stages_status_check;
ALTER TABLE job_stages RENAME CONSTRAINT application_stages_pkey TO job_stages_pkey;

-- Partial: every consumer filters applied_at IS NOT NULL (saved cards excluded).
CREATE INDEX idx_jobs_applied_at ON jobs(applied_at) WHERE applied_at IS NOT NULL;
CREATE INDEX idx_jobs_resume_id ON jobs(resume_id);
CREATE INDEX idx_jobs_resume_builder_id ON jobs(resume_builder_id) WHERE resume_builder_id IS NOT NULL;
CREATE INDEX idx_jobs_current_stage_id ON jobs(current_stage_id) WHERE current_stage_id IS NOT NULL;
-- Replaces the dropped idx_jobs_board_column(user_id, board_column): the board
-- list filters user_id + status, analytics/companies filter user_id + applied_at.
CREATE INDEX idx_jobs_user_status ON jobs(user_id, status);
CREATE INDEX idx_jobs_user_applied ON jobs(user_id) WHERE applied_at IS NOT NULL;

-- (12) re-enable triggers
ALTER TABLE jobs      ENABLE TRIGGER update_jobs_updated_at;
ALTER TABLE comments  ENABLE TRIGGER update_comments_updated_at;
ALTER TABLE reminders ENABLE TRIGGER update_reminders_updated_at;

COMMIT;
