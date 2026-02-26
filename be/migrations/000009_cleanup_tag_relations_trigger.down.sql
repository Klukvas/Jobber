DROP TRIGGER IF EXISTS trg_cleanup_tag_relations_applications ON applications;
DROP TRIGGER IF EXISTS trg_cleanup_tag_relations_jobs ON jobs;
DROP TRIGGER IF EXISTS trg_cleanup_tag_relations_companies ON companies;
DROP FUNCTION IF EXISTS cleanup_tag_relations();
