-- Keep writer exclusion, state checks and all drops in one transaction, including
-- migration runners that send SQL statements separately.
BEGIN ISOLATION LEVEL READ COMMITTED;
LOCK TABLE ai_provider_budget, credit_grants, ai_credit_operations,
 ai_credit_attempts, credit_payment_holds IN ACCESS EXCLUSIVE MODE;
-- Financial identities, spend and safety blocks must survive every restart/rollback.
-- Only a completely unused ledger can be removed.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM credit_grants)
 OR EXISTS(SELECT 1 FROM credit_payment_holds)
 OR EXISTS(SELECT 1 FROM ai_credit_operations)
 OR EXISTS(SELECT 1 FROM ai_credit_attempts)
 OR EXISTS(SELECT 1 FROM ai_provider_budget WHERE allocated_nanodollars>0 OR blocked) THEN
  RAISE EXCEPTION 'cannot roll back a used AI credit ledger or provider safety block';
 END IF;
END $$;
DROP TABLE ai_credit_attempts;
DROP TABLE ai_provider_budget;
DROP TABLE ai_credit_operations;
DROP TABLE credit_payment_holds;
DROP TABLE credit_grants;
COMMIT;
