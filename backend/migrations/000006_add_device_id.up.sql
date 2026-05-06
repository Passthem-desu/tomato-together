ALTER TABLE refresh_tokens ADD COLUMN device_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_device_id ON refresh_tokens(device_id);