-- Populate empty application names with job titles
UPDATE applications 
SET name = jobs.title
FROM jobs
WHERE applications.job_id = jobs.id 
AND (applications.name = '' OR applications.name IS NULL);
