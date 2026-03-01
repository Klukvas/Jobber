CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    paddle_subscription_id TEXT UNIQUE,
    paddle_customer_id TEXT,
    status TEXT NOT NULL DEFAULT 'free',
    plan TEXT NOT NULL DEFAULT 'free',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_paddle_sub_id ON subscriptions(paddle_subscription_id);

-- Create a 'free' subscription row for every existing user
INSERT INTO subscriptions (user_id, status, plan)
SELECT id, 'free', 'free' FROM users
ON CONFLICT (user_id) DO NOTHING;
