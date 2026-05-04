-- 000001_init_schema.down.sql
-- TomatoTogether 初始数据库 Schema 回滚

DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS user_statuses;
DROP TABLE IF EXISTS pomodoro_sessions;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS room_tokens;
DROP TABLE IF EXISTS room_members;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS users;