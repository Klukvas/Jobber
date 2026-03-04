-- Revert: restore NOT NULL constraint and ON DELETE RESTRICT FK
-- Remove applications that have no resume (NULL resume_id) before restoring NOT NULL
DELETE FROM applications WHERE resume_id IS NULL;
ALTER TABLE applications DROP CONSTRAINT applications_resume_id_fkey;
ALTER TABLE applications ADD CONSTRAINT applications_resume_id_fkey
    FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE RESTRICT;
ALTER TABLE applications ALTER COLUMN resume_id SET NOT NULL;
