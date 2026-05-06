-- Migration 000004: JWT refresh tokens + multi-device support
-- 1. Create refresh_tokens table for JWT refresh token storage
-- 2. Remove UNIQUE(member_id, room_id) constraint from room_tokens to allow multi-device
-- This migration is wrapped in a single transaction.

-- ============================================================
-- Part 1: Create refresh_tokens table
-- ============================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    device_name TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_member_id ON refresh_tokens(member_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- ============================================================
-- Part 2: Rebuild room_tokens — remove UNIQUE(member_id, room_id)
-- SQLite does not support ALTER TABLE DROP CONSTRAINT.
-- We recreate the table without the unique constraint.
-- Preserve all existing columns including token_hash from 000003.
-- ============================================================
CREATE TABLE room_tokens_new (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    token_hash TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

INSERT INTO room_tokens_new 
    (id, member_id, room_id, token, token_hash, created_at, expires_at, last_heartbeat)
SELECT 
    id, member_id, room_id, token, COALESCE(token_hash, ''), 
    created_at, expires_at, last_heartbeat
FROM room_tokens;

DROP TABLE room_tokens;
ALTER TABLE room_tokens_new RENAME TO room_tokens;

-- Rebuild indices
CREATE INDEX IF NOT EXISTS idx_room_tokens_token_hash ON room_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_room_tokens_member_id ON room_tokens(member_id);
CREATE INDEX IF NOT EXISTS idx_room_tokens_room_id ON room_tokens(room_id);
