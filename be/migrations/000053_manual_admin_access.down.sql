BEGIN ISOLATION LEVEL READ COMMITTED;
-- Rollback is safe before use. Never destroy allowance/audit or AI liability.
LOCK TABLE credit_grants, manual_access_grants IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM credit_grants WHERE kind='manual_pro')
 OR EXISTS(SELECT 1 FROM manual_access_grants) THEN
  RAISE EXCEPTION 'Manual access history exists; retain schema and roll back application code instead';
 END IF;
END $$;
DROP TABLE manual_access_grants;
ALTER TABLE credit_grants DROP CONSTRAINT credit_grants_kind_check;
ALTER TABLE credit_grants DROP CONSTRAINT credit_grants_check;
ALTER TABLE credit_grants ADD CONSTRAINT credit_grants_kind_check CHECK(kind IN ('free_once','pro'));
ALTER TABLE credit_grants ADD CONSTRAINT credit_grants_check CHECK(
 (kind='free_once' AND credits=30 AND period_start IS NULL AND period_end IS NULL)
 OR (kind='pro' AND credits=500 AND provider='creem' AND mode IN ('live','test')
 AND transaction_id<>'' AND subscription_id<>'' AND customer_id<>'' AND product_id<>''
 AND period_start IS NOT NULL AND period_end>period_start));

COMMIT;
