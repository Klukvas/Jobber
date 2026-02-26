-- Trigger function: delete orphaned tag_relations when an entity is deleted
CREATE OR REPLACE FUNCTION cleanup_tag_relations()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM tag_relations
    WHERE entity_type = TG_ARGV[0]
      AND entity_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Clean up tag_relations when a company is deleted
CREATE TRIGGER trg_cleanup_tag_relations_companies
    BEFORE DELETE ON companies
    FOR EACH ROW EXECUTE FUNCTION cleanup_tag_relations('company');

-- Clean up tag_relations when a job is deleted
CREATE TRIGGER trg_cleanup_tag_relations_jobs
    BEFORE DELETE ON jobs
    FOR EACH ROW EXECUTE FUNCTION cleanup_tag_relations('job');

-- Clean up tag_relations when an application is deleted
CREATE TRIGGER trg_cleanup_tag_relations_applications
    BEFORE DELETE ON applications
    FOR EACH ROW EXECUTE FUNCTION cleanup_tag_relations('application');
