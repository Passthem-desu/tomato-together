-- Store planned break settings in pomodoro sessions
ALTER TABLE pomodoro_sessions ADD COLUMN planned_rest_duration INTEGER NOT NULL DEFAULT 300;
ALTER TABLE pomodoro_sessions ADD COLUMN planned_long_break_duration INTEGER NOT NULL DEFAULT 900;
ALTER TABLE pomodoro_sessions ADD COLUMN sessions_before_long_break INTEGER NOT NULL DEFAULT 4;
