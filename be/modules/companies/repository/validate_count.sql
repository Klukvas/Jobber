-- Validation query to check application counting for companies
-- This query helps verify the relationship chain: company -> job -> application

-- Query 1: Show all companies with their application counts (current logic)
WITH company_stats AS (
    SELECT 
        c.id,
        c.name,
        c.user_id,
        COALESCE(COUNT(DISTINCT a.id), 0) as applications_count,
        COALESCE(COUNT(DISTINCT a.id) FILTER (WHERE a.status = 'active'), 0) as active_applications_count
    FROM companies c
    LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
    LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = c.user_id
    WHERE c.user_id = $1
    GROUP BY c.id, c.name, c.user_id
)
SELECT * FROM company_stats;

-- Query 2: Validate the relationship chain step-by-step
-- This shows potential issues with NULL company_id in jobs
SELECT 
    c.id as company_id,
    c.name as company_name,
    c.user_id as company_user_id,
    j.id as job_id,
    j.title as job_title,
    j.user_id as job_user_id,
    j.company_id as job_company_id,
    a.id as application_id,
    a.user_id as application_user_id,
    a.status as application_status
FROM companies c
LEFT JOIN jobs j ON j.company_id = c.id
LEFT JOIN applications a ON a.job_id = j.id
WHERE c.user_id = $1
ORDER BY c.name, j.title;

-- Query 3: Check for orphaned applications (applications whose job doesn't have a company)
SELECT 
    a.id as application_id,
    a.user_id,
    j.id as job_id,
    j.title,
    j.company_id,
    CASE 
        WHEN j.company_id IS NULL THEN 'Job has no company'
        ELSE 'Job has company'
    END as issue
FROM applications a
JOIN jobs j ON a.job_id = j.id
WHERE a.user_id = $1;

-- Query 4: Count applications per company (verification)
SELECT 
    c.id,
    c.name,
    COUNT(DISTINCT j.id) as jobs_count,
    COUNT(DISTINCT a.id) as applications_count
FROM companies c
LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
LEFT JOIN applications a ON a.job_id = j.id
WHERE c.user_id = $1
GROUP BY c.id, c.name
ORDER BY c.name;

-- Query 5: CORRECTED - Check if we need to remove redundant user_id check on applications
-- The applications user_id should match because it references the job, which references the company
-- But we still need it for security to ensure multi-tenancy
WITH company_stats AS (
    SELECT 
        c.id,
        c.name,
        COALESCE(COUNT(DISTINCT a.id), 0) as applications_count_with_check,
        COALESCE(COUNT(DISTINCT a2.id), 0) as applications_count_without_check
    FROM companies c
    LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
    LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = c.user_id
    LEFT JOIN applications a2 ON a2.job_id = j.id
    WHERE c.user_id = $1
    GROUP BY c.id, c.name
)
SELECT * FROM company_stats
WHERE applications_count_with_check != applications_count_without_check;
