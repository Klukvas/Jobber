-- Nothing to reverse: the up migration changes no schema and no data.
--
-- This is a no-op, not a restore. Values cleared by the earlier, destructive
-- version of 000046 cannot be recovered here — see the note in the up file and
-- the pre-deploy dumps in ~/jobber/backups.
SELECT 1;
