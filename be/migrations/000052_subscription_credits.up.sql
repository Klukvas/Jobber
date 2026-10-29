-- Credit principals and payment identities are durable; provider liability has no TTL.
CREATE TABLE credit_grants (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind TEXT NOT NULL CHECK(kind IN ('free_once','pro')),
 credits INTEGER NOT NULL CHECK(credits > 0),
 spend_limit_nanodollars BIGINT GENERATED ALWAYS AS (CASE WHEN kind='pro' THEN 5000000000::bigint ELSE 3000000000::bigint END) STORED,
 provider TEXT NOT NULL DEFAULT '',
 mode TEXT NOT NULL DEFAULT '',
 transaction_id TEXT NOT NULL DEFAULT '',
 subscription_id TEXT NOT NULL DEFAULT '',
 customer_id TEXT NOT NULL DEFAULT '',
 product_id TEXT NOT NULL DEFAULT '',
 period_start TIMESTAMPTZ,
 period_end TIMESTAMPTZ,
 revoked_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK((kind='free_once' AND credits=30 AND period_start IS NULL AND period_end IS NULL)
 OR (kind='pro' AND credits=500 AND provider='creem' AND mode IN ('live','test')
 AND transaction_id<>'' AND subscription_id<>'' AND customer_id<>'' AND product_id<>''
 AND period_start IS NOT NULL AND period_end>period_start))
);
CREATE UNIQUE INDEX credit_grants_free_once ON credit_grants(user_id) WHERE kind='free_once';
CREATE UNIQUE INDEX credit_grants_transaction ON credit_grants(provider,mode,transaction_id) WHERE kind='pro';
CREATE UNIQUE INDEX credit_grants_period ON credit_grants(provider,mode,subscription_id,period_start,period_end) WHERE kind='pro';
CREATE INDEX credit_grants_owner ON credit_grants(user_id,period_end);

CREATE TABLE credit_payment_holds (
 provider TEXT NOT NULL,
 mode TEXT NOT NULL,
 transaction_id TEXT NOT NULL,
 reason TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(provider,mode,transaction_id)
);

CREATE TABLE ai_credit_operations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 operation_key TEXT NOT NULL,
 kind TEXT NOT NULL,
 fingerprint TEXT NOT NULL,
 grant_id UUID NOT NULL REFERENCES credit_grants(id) ON DELETE CASCADE,
 credits INTEGER NOT NULL CHECK(credits>0),
 state TEXT NOT NULL DEFAULT 'reserved' CHECK(state IN ('reserved','dispatched','settled','released')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(user_id,operation_key)
);
CREATE INDEX ai_credit_operations_grant ON ai_credit_operations(grant_id) WHERE state<>'released';

-- This is a lifetime application cap, not an automatically replenishing allowance.
-- No row is automatically funded: startup config must explicitly supply a finite cap.
CREATE TABLE ai_provider_budget (
 provider TEXT PRIMARY KEY,
 limit_nanodollars BIGINT NOT NULL CHECK(limit_nanodollars>=0),
 blocked BOOLEAN NOT NULL DEFAULT false,
 block_reason TEXT NOT NULL DEFAULT '',
 allocated_nanodollars BIGINT NOT NULL DEFAULT 0 CHECK(allocated_nanodollars>=0)
);
CREATE TABLE ai_credit_attempts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 operation_id UUID NOT NULL REFERENCES ai_credit_operations(id) ON DELETE CASCADE,
 stage TEXT NOT NULL,
 model TEXT NOT NULL,
 pricing_version TEXT NOT NULL DEFAULT 'anthropic-2026-10-25',
 provider TEXT NOT NULL REFERENCES ai_provider_budget(provider),
 reserved_nanodollars BIGINT NOT NULL CHECK(reserved_nanodollars>0),
 actual_nanodollars BIGINT CHECK(actual_nanodollars>=0 AND actual_nanodollars<=reserved_nanodollars),
 state TEXT NOT NULL CHECK(state IN ('reserved','dispatched','settled','released')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 dispatched_at TIMESTAMPTZ,
 settled_at TIMESTAMPTZ,
 UNIQUE(operation_id,stage)
);
