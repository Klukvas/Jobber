-- The comments updated_at trigger fired on every UPDATE, including the
-- ON DELETE SET NULL that Postgres runs on comments.stage_id when a pipeline
-- stage is deleted. That bumped updated_at on a comment nobody edited, so the UI
-- showed it as "edited". Only a change to the comment text counts as an edit.
BEGIN;

DROP TRIGGER IF EXISTS update_comments_updated_at ON comments;
CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content)
    EXECUTE FUNCTION update_updated_at_column();

COMMIT;
