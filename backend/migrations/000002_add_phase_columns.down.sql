-- Revert: remove phase state machine columns
ALTER TABLE pomodoro_sessions DROP COLUMN paused_at;
ALTER TABLE pomodoro_sessions DROP COLUMN rest_duration;
ALTER TABLE pomodoro_sessions DROP COLUMN is_long_break;
