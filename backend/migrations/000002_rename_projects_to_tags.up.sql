-- 000002_rename_projects_to_tags.up.sql
-- Safe: uses only ALTER TABLE RENAME (no table recreation, no data loss risk)
-- SQLite 3.25+ required (Docker images all support this)

-- 1. Rename table
ALTER TABLE projects RENAME TO tags;

-- 2. Rename FK columns in dependent tables
ALTER TABLE tasks RENAME COLUMN project_id TO tag_id;
ALTER TABLE pomodoro_sessions RENAME COLUMN project_id TO tag_id;
