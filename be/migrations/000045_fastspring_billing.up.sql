-- Migrate billing columns from Paddle-specific to provider-neutral names and
-- add the lifecycle-ordering guard the FastSpring webhook handler relies on.
--
-- Data safety: there are no paying subscribers at migration time — every
-- existing row is a free plan with NULL paddle_subscription_id /
-- paddle_customer_id. The renames are therefore metadata-only, and the new
-- partial UNIQUE index cannot collide because NULLs never conflict. That WHERE
-- clause is what replaces the deduplication step a UNIQUE index over populated
-- data would need.
--
-- Index creation is deliberately not CONCURRENT: golang-migrate wraps each file
-- in a transaction (CONCURRENTLY is illegal there), and `subscriptions` holds
-- one small row per user, so the lock is momentary.

ALTER TABLE subscriptions RENAME COLUMN paddle_subscription_id TO external_subscription_id;
ALTER TABLE subscriptions RENAME COLUMN paddle_customer_id TO external_account_id;

ALTER INDEX IF EXISTS idx_subscriptions_paddle_sub_id RENAME TO idx_subscriptions_external_sub_id;

-- Keep the auto-generated UNIQUE constraint index in step with its column name.
ALTER INDEX IF EXISTS subscriptions_paddle_subscription_id_key
    RENAME TO subscriptions_external_subscription_id_key;

-- The provider account is the only server-side link between a purchase and a
-- local user, so it must resolve to exactly one user. A collision fails the
-- checkout rather than risking a mis-grant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_external_account_id
    ON subscriptions (external_account_id)
    WHERE external_account_id IS NOT NULL;

-- The provider's own timestamp for the newest subscription *state change*
-- applied to this row — the payload's `data.changed`, not when the webhook was
-- created or delivered. (Charge events carry no `changed`, so those fall back to
-- the envelope's `created`.) That distinction is the whole point: a manual
-- resend arrives under a fresh event ID *and* a fresh envelope timestamp, so
-- ordering on delivery time would let it slip past event-ID de-duplication and
-- resurrect a deactivated subscription. Writes are refused unless strictly
-- newer than this value.
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS last_event_at TIMESTAMPTZ;
