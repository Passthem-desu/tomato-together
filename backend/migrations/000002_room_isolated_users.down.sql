-- 000002_room_isolated_users.down.sql
-- 回滚：移除房间隔离用户设计，恢复到独立 users 表（v1.0）

-- 删除新增的表
DROP TABLE IF EXISTS announcements;
DROP TABLE IF EXISTS user_statuses;
DROP TABLE IF EXISTS pomodoro_sessions;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS room_tokens;
DROP TABLE IF EXISTS room_members;
-- rooms 表保留，但可能需要清理字段

-- =============================================
-- 恢复 v1.0 的用户表设计
-- =============================================

-- 用户表（v1.0 全局用户）
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);

-- 房间成员表（v1.0，关联全局用户）
CREATE TABLE IF NOT EXISTS room_members (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    is_owner INTEGER NOT NULL DEFAULT 0,
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (room_id, user_id)
);

CREATE INDEX idx_room_members_room_id ON room_members(room_id);
CREATE INDEX idx_room_members_user_id ON room_members(user_id);

-- RoomToken 表（v1.0，关联 user_id）
CREATE TABLE IF NOT EXISTS room_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (user_id, room_id)
);

CREATE INDEX idx_room_tokens_token ON room_tokens(token);
CREATE INDEX idx_room_tokens_user_id ON room_tokens(user_id);
CREATE INDEX idx_room_tokens_room_id ON room_tokens(room_id);
CREATE INDEX idx_room_tokens_expires_at ON room_tokens(expires_at);

-- 项目表（v1.0，关联 user_id）
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_projects_user_id ON projects(user_id);

-- WIP 待办表（v1.0，关联 user_id）
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    client_id TEXT UNIQUE,
    user_id TEXT NOT NULL,
    project_id TEXT,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'TODO' CHECK(status IN ('TODO', 'WIP', 'DONE')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_client_id ON tasks(client_id);

-- 番茄记录表（v1.0，关联 user_id 和 leader_id）
CREATE TABLE IF NOT EXISTS pomodoro_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    project_id TEXT,
    task_id TEXT,
    duration INTEGER NOT NULL DEFAULT 0,
    planned_duration INTEGER NOT NULL DEFAULT 1500,
    is_followed INTEGER NOT NULL DEFAULT 0,
    leader_id TEXT,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL,
    FOREIGN KEY (leader_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_pomodoro_sessions_user_id ON pomodoro_sessions(user_id);
CREATE INDEX idx_pomodoro_sessions_room_id ON pomodoro_sessions(room_id);
CREATE INDEX idx_pomodoro_sessions_started_at ON pomodoro_sessions(started_at);

-- 用户状态表（v1.0，关联 user_id）
CREATE TABLE IF NOT EXISTS user_statuses (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    emoji TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (user_id, room_id)
);

CREATE INDEX idx_user_statuses_user_room ON user_statuses(user_id, room_id);

-- 公告表（v1.0，关联 user_id）
CREATE TABLE IF NOT EXISTS announcements (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_announcements_room_id ON announcements(room_id);
CREATE INDEX idx_announcements_created_at ON announcements(created_at DESC);