-- Billing moved from FastSpring to Creem. The subscription columns are already
-- provider-neutral (migration 000045), so there is no schema change: only values.
--
-- Data safety: there were no paying FastSpring subscribers at cut-over, so every
-- row holding a FastSpring identifier is a test-mode leftover. Left in place,
-- those identifiers would be sent to Creem on cancel / change-plan / portal and
-- fail there, and a stale external_subscription_id would make the link guard
-- refuse the user's first real Creem subscription. Reset them to the free plan.
--
-- The WHERE clause keeps the migration from touching anything that already looks
-- like Creem: its ids are prefixed `sub_` and `cust_`, FastSpring's are opaque
-- strings without those prefixes. Re-running it, or applying it to a database
-- that already holds Creem subscriptions (a staging copy, say), therefore leaves
-- those rows alone instead of downgrading paying users. (A leftover Paddle
-- subscription id, `sub_...`, is also kept; there were none, see ADR-0002.)
--
-- webhook_events needs no cleanup: it only de-duplicates event IDs, and Creem's
-- `evt_` IDs cannot collide with FastSpring's.
UPDATE subscriptions
SET external_subscription_id = NULL,
    external_account_id      = NULL,
    status                   = 'free',
    plan                     = 'free',
    current_period_start     = NULL,
    current_period_end       = NULL,
    cancel_at                = NULL,
    last_event_at            = NULL,
    updated_at               = NOW()
WHERE (external_subscription_id IS NOT NULL AND external_subscription_id NOT LIKE 'sub\_%')
   OR (external_account_id IS NOT NULL AND external_account_id NOT LIKE 'cust\_%');
