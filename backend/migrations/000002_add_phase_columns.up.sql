-- Add phase state machine columns to pomodoro_sessions
ALTER TABLE pomodoro_sessions ADD COLUMN paused_at DATETIME;
ALTER TABLE pomodoro_sessions ADD COLUMN rest_duration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE pomodoro_sessions ADD COLUMN is_long_break INTEGER NOT NULL DEFAULT 0;
