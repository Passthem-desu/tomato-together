-- Rollback: 000004_jwt_refresh_tokens
-- 1. Drop refresh_tokens table
-- 2. Restore UNIQUE(member_id, room_id) constraint on room_tokens

DROP TABLE IF EXISTS refresh_tokens;

CREATE TABLE room_tokens_old (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (member_id, room_id)
);

INSERT INTO room_tokens_old SELECT * FROM room_tokens;
DROP TABLE room_tokens;
ALTER TABLE room_tokens_old RENAME TO room_tokens;

CREATE INDEX IF NOT EXISTS idx_room_tokens_token_hash ON room_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_room_tokens_member_id ON room_tokens(member_id);
CREATE INDEX IF NOT EXISTS idx_room_tokens_room_id ON room_tokens(room_id);
