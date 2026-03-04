-- Allow applications to exist without a resume (resume_id becomes nullable)
-- When a resume is deleted, applications referencing it will have resume_id set to NULL
ALTER TABLE applications DROP CONSTRAINT applications_resume_id_fkey;
ALTER TABLE applications ALTER COLUMN resume_id DROP NOT NULL;
ALTER TABLE applications ADD CONSTRAINT applications_resume_id_fkey
    FOREIGN KEY (resume_id) REFERENCES resumes(id) ON DELETE SET NULL;
