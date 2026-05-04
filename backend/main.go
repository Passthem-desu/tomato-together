package main

import (
	"database/sql"
	"log"
	"os"

	"tomatogether/backend/internal/api"
	"tomatogether/backend/internal/repository"
	"tomatogether/backend/internal/service"
	"tomatogether/backend/internal/sse"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// 初始化数据库
	dbPath := getEnv("DB_PATH", "./data/tomatogether.db")
	if err := os.MkdirAll(getDir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// 初始化数据库表
	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// 初始化依赖
	userRepo := repository.NewUserRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	roomMemberRepo := repository.NewRoomMemberRepository(db)
	roomTokenRepo := repository.NewRoomTokenRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	pomodoroRepo := repository.NewPomodoroRepository(db)
	statusRepo := repository.NewUserStatusRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)

	userSvc := service.NewUserService(userRepo, roomTokenRepo)
	roomSvc := service.NewRoomService(roomRepo, roomMemberRepo, userRepo)
	authSvc := service.NewAuthService(userRepo, roomTokenRepo)
	pomodoroSvc := service.NewPomodoroService(pomodoroRepo, roomMemberRepo)
	projectSvc := service.NewProjectService(projectRepo)
	taskSvc := service.NewTaskService(taskRepo, projectRepo)
	statusSvc := service.NewUserStatusService(statusRepo)
	announcementSvc := service.NewAnnouncementService(announcementRepo, roomRepo, roomMemberRepo)

	// SSE Hub
	sseHub := sse.NewHub()
	go sseHub.Run()

	// API Server
	port := getEnv("PORT", "8080")
	server := api.NewServer(port, db, sseHub)
	server.RegisterServices(
		userSvc, roomSvc, authSvc, pomodoroSvc,
		projectSvc, taskSvc, statusSvc, announcementSvc,
	)
	server.RegisterRepositories(
		userRepo, roomRepo, roomMemberRepo, roomTokenRepo,
		projectRepo, taskRepo, pomodoroRepo, statusRepo, announcementRepo,
	)

	log.Printf("Server starting on port %s...", port)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func initSchema(db *sql.DB) error {
	schema := `
	-- 用户表
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	-- 用户名索引
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

	-- 房间表
	CREATE TABLE IF NOT EXISTS rooms (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL DEFAULT '',
		is_readonly INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_rooms_name ON rooms(name);

	-- 房间成员表
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

	CREATE INDEX IF NOT EXISTS idx_room_members_room_id ON room_members(room_id);
	CREATE INDEX IF NOT EXISTS idx_room_members_user_id ON room_members(user_id);

	-- RoomToken 表（L2 认证）
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

	CREATE INDEX IF NOT EXISTS idx_room_tokens_token ON room_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_room_tokens_user_id ON room_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_room_tokens_room_id ON room_tokens(room_id);
	CREATE INDEX IF NOT EXISTS idx_room_tokens_expires_at ON room_tokens(expires_at);

	-- 项目表
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);

	-- WIP 待办表（Task）
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

	CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks(user_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
	CREATE INDEX IF NOT EXISTS idx_tasks_client_id ON tasks(client_id);

	-- 番茄记录表
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

	CREATE INDEX IF NOT EXISTS idx_pomodoro_sessions_user_id ON pomodoro_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_pomodoro_sessions_room_id ON pomodoro_sessions(room_id);
	CREATE INDEX IF NOT EXISTS idx_pomodoro_sessions_started_at ON pomodoro_sessions(started_at);

	-- 用户状态表
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

	CREATE INDEX IF NOT EXISTS idx_user_statuses_user_room ON user_statuses(user_id, room_id);

	-- 公告表（仅公告持久化）
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

	CREATE INDEX IF NOT EXISTS idx_announcements_room_id ON announcements(room_id);
	CREATE INDEX IF NOT EXISTS idx_announcements_created_at ON announcements(created_at DESC);
	`
	_, err := db.Exec(schema)
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}