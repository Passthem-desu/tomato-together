-- 000002_room_isolated_users.up.sql
-- TomatoTogether v2.0 房间隔离用户设计
-- 核心变更：废除 users 表，room_members 成为核心用户表

-- =============================================
-- 第一部分：创建新表（房间隔离用户）
-- =============================================

-- 房间表（保持不变，增加一些字段）
CREATE TABLE IF NOT EXISTS rooms (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL DEFAULT '',
    is_readonly INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rooms_name ON rooms(name);

-- 房间成员表（核心用户表，替代原来的 users 表）
-- 用户名在同一房间内唯一，不同房间可以有相同的用户名
CREATE TABLE IF NOT EXISTS room_members (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    is_owner INTEGER NOT NULL DEFAULT 0,
    joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (room_id, username)
);

CREATE INDEX idx_room_members_room_id ON room_members(room_id);
CREATE INDEX idx_room_members_username ON room_members(room_id, username);

-- RoomToken 表（L2 认证，关联 member_id）
CREATE TABLE IF NOT EXISTS room_tokens (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    last_heartbeat DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (member_id, room_id)
);

CREATE INDEX idx_room_tokens_token ON room_tokens(token);
CREATE INDEX idx_room_tokens_member_id ON room_tokens(member_id);
CREATE INDEX idx_room_tokens_room_id ON room_tokens(room_id);
CREATE INDEX idx_room_tokens_expires_at ON room_tokens(expires_at);

-- =============================================
-- 第二部分：项目和任务表（关联 member_id 和 room_id）
-- =============================================

-- 项目表（按房间隔离）
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (member_id, room_id, name)
);

CREATE INDEX idx_projects_member_id ON projects(member_id);
CREATE INDEX idx_projects_room_id ON projects(room_id);

-- WIP 待办表（按房间隔离）
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    client_id TEXT,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    project_id TEXT,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'TODO' CHECK(status IN ('TODO', 'WIP', 'DONE')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

CREATE INDEX idx_tasks_member_id ON tasks(member_id);
CREATE INDEX idx_tasks_room_id ON tasks(room_id);
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_client_id ON tasks(client_id);

-- =============================================
-- 第三部分：番茄记录（关联 member_id 和 room_id）
-- =============================================

-- 番茄记录表
CREATE TABLE IF NOT EXISTS pomodoro_sessions (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    project_id TEXT,
    task_id TEXT,
    duration INTEGER NOT NULL DEFAULT 0,
    planned_duration INTEGER NOT NULL DEFAULT 1500,
    is_followed INTEGER NOT NULL DEFAULT 0,
    leader_id TEXT,
    started_at DATETIME NOT NULL,
    ended_at DATETIME,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE SET NULL,
    FOREIGN KEY (leader_id) REFERENCES room_members(id) ON DELETE SET NULL
);

CREATE INDEX idx_pomodoro_sessions_member_id ON pomodoro_sessions(member_id);
CREATE INDEX idx_pomodoro_sessions_room_id ON pomodoro_sessions(room_id);
CREATE INDEX idx_pomodoro_sessions_started_at ON pomodoro_sessions(started_at);

-- =============================================
-- 第四部分：状态和公告（关联 member_id 和 room_id）
-- =============================================

-- 用户状态表
CREATE TABLE IF NOT EXISTS user_statuses (
    id TEXT PRIMARY KEY,
    member_id TEXT NOT NULL,
    room_id TEXT NOT NULL,
    emoji TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (member_id) REFERENCES room_members(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    UNIQUE (member_id, room_id)
);

CREATE INDEX idx_user_statuses_member_room ON user_statuses(member_id, room_id);

-- 公告表
CREATE TABLE IF NOT EXISTS announcements (
    id TEXT PRIMARY KEY,
    room_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES room_members(id) ON DELETE CASCADE
);

CREATE INDEX idx_announcements_room_id ON announcements(room_id);
CREATE INDEX idx_announcements_created_at ON announcements(created_at DESC);