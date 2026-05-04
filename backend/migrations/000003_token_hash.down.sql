DROP TABLE IF EXISTS token_revocations;

-- SQLite does not support DROP COLUMN; recreate table without token_hash
-- This is a no-op for down migration since we only added columns
