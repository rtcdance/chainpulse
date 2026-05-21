-- Webhooks table for event-driven notifications
CREATE TABLE IF NOT EXISTS webhooks (
    id VARCHAR(255) PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events JSONB DEFAULT '["event:created","event:confirmed","event:failed"]',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_delivery_at TIMESTAMP,
    last_delivery_status VARCHAR(32),
    failure_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_webhooks_client_id ON webhooks(client_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_enabled ON webhooks(enabled);
