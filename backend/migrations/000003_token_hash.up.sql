-- Add token_hash column for secure token storage
ALTER TABLE room_tokens ADD COLUMN token_hash TEXT DEFAULT '';

-- Create token_revocations table for token blacklist
CREATE TABLE IF NOT EXISTS token_revocations (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL,
    reason TEXT DEFAULT '',
    revoked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_token_revocations_hash ON token_revocations(token_hash);
CREATE INDEX IF NOT EXISTS idx_token_revocations_at ON token_revocations(revoked_at);
