BEGIN;
LOCK TABLE ai_credit_operations IN ACCESS EXCLUSIVE MODE;
-- The old application would charge refunded provider calls again on rollback.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM ai_credit_operations WHERE charge_state<>'legacy' AND state<>'released') THEN
  RAISE EXCEPTION 'cannot remove result charge accounting from used AI credit ledger';
 END IF;
END $$;
DROP INDEX ai_credit_operations_charge_grant;
ALTER TABLE ai_credit_operations DROP COLUMN charge_finished_at, DROP COLUMN charge_reason, DROP COLUMN charge_state;
COMMIT;
