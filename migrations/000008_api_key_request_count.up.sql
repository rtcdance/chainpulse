-- Add request_count column to api_keys for usage tracking
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS request_count BIGINT DEFAULT 0;