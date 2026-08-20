-- Reverse 000042. Best-effort: the notes column shape is restored, but the
-- backfilled comments are NOT deleted or folded back — they are indistinguishable
-- from genuine notes-era comments, and dropping them would lose real user data.
-- Prod recovery of the original layout is the pre-deploy dump.
BEGIN;

ALTER TABLE jobs ADD COLUMN IF NOT EXISTS notes TEXT;

COMMIT;
