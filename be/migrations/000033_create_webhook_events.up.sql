CREATE TABLE IF NOT EXISTS webhook_events (
    event_id  TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_processed_at ON webhook_events (processed_at);
