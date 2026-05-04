-- 000002_rename_projects_to_tags.down.sql
-- Undo: rename back

ALTER TABLE pomodoro_sessions RENAME COLUMN tag_id TO project_id;
ALTER TABLE tasks RENAME COLUMN tag_id TO project_id;
ALTER TABLE tags RENAME TO projects;
