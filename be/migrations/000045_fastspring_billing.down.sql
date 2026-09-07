ALTER TABLE subscriptions DROP COLUMN IF EXISTS last_event_at;

DROP INDEX IF EXISTS idx_subscriptions_external_account_id;

ALTER INDEX IF EXISTS subscriptions_external_subscription_id_key
    RENAME TO subscriptions_paddle_subscription_id_key;

ALTER INDEX IF EXISTS idx_subscriptions_external_sub_id RENAME TO idx_subscriptions_paddle_sub_id;

ALTER TABLE subscriptions RENAME COLUMN external_account_id TO paddle_customer_id;
ALTER TABLE subscriptions RENAME COLUMN external_subscription_id TO paddle_subscription_id;
