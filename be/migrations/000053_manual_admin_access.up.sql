BEGIN ISOLATION LEVEL READ COMMITTED;
-- Add a separate manual allowance; never manufacture a Creem payment identity.
ALTER TABLE credit_grants DROP CONSTRAINT credit_grants_kind_check;
ALTER TABLE credit_grants DROP CONSTRAINT credit_grants_check;
ALTER TABLE credit_grants ADD CONSTRAINT credit_grants_kind_check CHECK(kind IN ('free_once','pro','manual_pro'));
ALTER TABLE credit_grants ADD CONSTRAINT credit_grants_check CHECK(
 (kind='free_once' AND credits=30 AND period_start IS NULL AND period_end IS NULL)
 OR (kind='pro' AND credits=500 AND provider='creem' AND mode IN ('live','test')
 AND transaction_id<>'' AND subscription_id<>'' AND customer_id<>'' AND product_id<>''
 AND period_start IS NOT NULL AND period_end>period_start)
 OR (kind='manual_pro' AND credits=500 AND provider='' AND mode='' AND transaction_id=''
 AND subscription_id='' AND customer_id='' AND product_id=''
 AND period_start IS NOT NULL AND period_end>period_start AND period_end<=period_start+interval '31 days'));
-- manual_pro intentionally retains the existing generated $3 provider spend cap.
CREATE TABLE manual_access_grants (
 grant_id UUID PRIMARY KEY REFERENCES credit_grants(id),
 actor_id UUID NOT NULL REFERENCES users(id),
 reason TEXT NOT NULL CHECK(char_length(reason) BETWEEN 3 AND 500),
 revoked_by UUID REFERENCES users(id),
 revoke_reason TEXT,
 CHECK ((revoked_by IS NULL AND revoke_reason IS NULL) OR
        (revoked_by IS NOT NULL AND char_length(revoke_reason) BETWEEN 3 AND 500))
);

COMMIT;
